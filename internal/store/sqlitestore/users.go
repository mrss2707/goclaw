//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/internal/store/base"
)

type SQLiteUserStore struct {
	db *sql.DB
}

func NewSQLiteUserStore(db *sql.DB) *SQLiteUserStore {
	return &SQLiteUserStore{db: db}
}

func (s *SQLiteUserStore) CreateUser(ctx context.Context, user *store.User) error {
	now := time.Now()
	user.ID = store.GenNewID()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.GoogleID, user.PasswordHash, user.DisplayName, user.AvatarURL,
		user.EmailVerified, user.VerificationCode, user.VerificationExp, user.Locale,
		base.JsonOrEmpty(user.Metadata), now, now,
	)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

var userSelectCols = `id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at`

func (s *SQLiteUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.GoogleID, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.EmailVerified, &u.VerificationCode, &u.VerificationExp, &u.Locale, &u.Metadata, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteUserStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.GoogleID, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.EmailVerified, &u.VerificationCode, &u.VerificationExp, &u.Locale, &u.Metadata, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteUserStore) GetUserByGoogleID(ctx context.Context, googleID string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE google_id = ?`, googleID,
	).Scan(&u.ID, &u.Email, &u.GoogleID, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.EmailVerified, &u.VerificationCode, &u.VerificationExp, &u.Locale, &u.Metadata, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteUserStore) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return execMapUpdate(ctx, s.db, "users", id, updates)
}

func (s *SQLiteUserStore) SetVerificationCode(ctx context.Context, id uuid.UUID, code string, exp time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET verification_code = ?, verification_exp = ?, updated_at = ? WHERE id = ?`,
		code, exp, time.Now(), id)
	return err
}

func (s *SQLiteUserStore) VerifyEmail(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET email_verified = 1, verification_code = NULL, verification_exp = NULL, updated_at = ? WHERE id = ?`,
		time.Now(), id)
	return err
}

func (s *SQLiteUserStore) LinkGoogleID(ctx context.Context, id uuid.UUID, googleID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET google_id = ?, updated_at = ? WHERE id = ?`,
		googleID, time.Now(), id)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

func (s *SQLiteUserStore) UnlinkGoogleID(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET google_id = NULL, updated_at = ? WHERE id = ?`,
		time.Now(), id)
	return err
}

func (s *SQLiteUserStore) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, time.Now(), id)
	return err
}

func (s *SQLiteUserStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM users WHERE id = ?`, id)
	return err
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "23505")
}
