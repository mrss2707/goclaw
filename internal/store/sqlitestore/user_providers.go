//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/crypto"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/internal/store/base"
)

type SQLiteUserProviderStore struct {
	db     *sql.DB
	encKey string
}

func NewSQLiteUserProviderStore(db *sql.DB, encryptionKey string) *SQLiteUserProviderStore {
	return &SQLiteUserProviderStore{db: db, encKey: encryptionKey}
}

func (s *SQLiteUserProviderStore) CreateProvider(ctx context.Context, p *store.UserProvider) error {
	now := time.Now()
	p.ID = store.GenNewID()
	p.CreatedAt = now
	p.UpdatedAt = now

	apiKey := p.APIKey
	if s.encKey != "" && apiKey != "" {
		encrypted, err := crypto.Encrypt(apiKey, s.encKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		apiKey = encrypted
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_providers (id, user_id, name, display_name, provider_type, api_base, api_key, enabled, settings, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Name, p.DisplayName, p.ProviderType, p.APIBase, apiKey,
		p.Enabled, base.JsonOrEmpty(p.Settings), p.CreatedAt, p.UpdatedAt,
	)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

func (s *SQLiteUserProviderStore) GetProvider(ctx context.Context, id uuid.UUID) (*store.UserProvider, error) {
	var p store.UserProvider
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, display_name, provider_type, api_base, api_key, enabled, settings, created_at, updated_at
		 FROM user_providers WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.DisplayName, &p.ProviderType, &p.APIBase, &p.APIKey,
		&p.Enabled, &p.Settings, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	p.APIKey = s.decryptKey(p.APIKey, p.Name)
	return &p, nil
}

func (s *SQLiteUserProviderStore) GetProviderByName(ctx context.Context, userID uuid.UUID, name string) (*store.UserProvider, error) {
	var p store.UserProvider
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, display_name, provider_type, api_base, api_key, enabled, settings, created_at, updated_at
		 FROM user_providers WHERE user_id = ? AND name = ?`, userID, name,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.DisplayName, &p.ProviderType, &p.APIBase, &p.APIKey,
		&p.Enabled, &p.Settings, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	p.APIKey = s.decryptKey(p.APIKey, p.Name)
	return &p, nil
}

func (s *SQLiteUserProviderStore) ListProviders(ctx context.Context, userID uuid.UUID) ([]store.UserProvider, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, display_name, provider_type, api_base, api_key, enabled, settings, created_at, updated_at
		 FROM user_providers WHERE user_id = ? ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []store.UserProvider
	for rows.Next() {
		var p store.UserProvider
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.DisplayName, &p.ProviderType, &p.APIBase, &p.APIKey,
			&p.Enabled, &p.Settings, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.APIKey = s.decryptKey(p.APIKey, p.Name)
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *SQLiteUserProviderStore) UpdateProvider(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	if apiKey, ok := updates["api_key"]; ok && s.encKey != "" {
		if keyStr, ok := apiKey.(string); ok && keyStr != "" {
			encrypted, err := crypto.Encrypt(keyStr, s.encKey)
			if err != nil {
				return fmt.Errorf("encrypt api key: %w", err)
			}
			updates["api_key"] = encrypted
		}
	}
	return execMapUpdate(ctx, s.db, "user_providers", id, updates)
}

func (s *SQLiteUserProviderStore) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_providers WHERE id = ?`, id)
	return err
}

func (s *SQLiteUserProviderStore) decryptKey(apiKey, name string) string {
	if s.encKey != "" && apiKey != "" {
		decrypted, err := crypto.Decrypt(apiKey, s.encKey)
		if err != nil {
			slog.Warn("failed to decrypt user provider API key", "provider", name, "error", err)
			return apiKey
		}
		return decrypted
	}
	return apiKey
}
