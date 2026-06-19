package pg

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

type PGUserStore struct {
	db *sql.DB
}

func NewPGUserStore(db *sql.DB) *PGUserStore {
	return &PGUserStore{db: db}
}

func (s *PGUserStore) CreateUser(ctx context.Context, user *store.User) error {
	now := time.Now()
	user.ID = store.GenNewID()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		user.ID, user.Email, user.GoogleID, user.PasswordHash, user.DisplayName, user.AvatarURL,
		user.EmailVerified, user.VerificationCode, user.VerificationExp, user.Locale,
		base.JsonOrEmpty(user.Metadata), user.CreatedAt, user.UpdatedAt,
	)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

func (s *PGUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at
		 FROM users WHERE id = $1`, id,
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

func (s *PGUserStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at
		 FROM users WHERE email = $1`, email,
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

func (s *PGUserStore) GetUserByGoogleID(ctx context.Context, googleID string) (*store.User, error) {
	var u store.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at
		 FROM users WHERE google_id = $1`, googleID,
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

func (s *PGUserStore) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return execMapUpdate(ctx, s.db, "users", id, updates)
}

func (s *PGUserStore) SetVerificationCode(ctx context.Context, id uuid.UUID, code string, exp time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET verification_code = $1, verification_exp = $2, updated_at = NOW() WHERE id = $3`,
		code, exp, id)
	return err
}

func (s *PGUserStore) VerifyEmail(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET email_verified = true, verification_code = NULL, verification_exp = NULL, updated_at = NOW() WHERE id = $1`,
		id)
	return err
}

func (s *PGUserStore) LinkGoogleID(ctx context.Context, id uuid.UUID, googleID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET google_id = $1, updated_at = NOW() WHERE id = $2`,
		googleID, id)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

func (s *PGUserStore) UnlinkGoogleID(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET google_id = NULL, updated_at = NOW() WHERE id = $1`,
		id)
	return err
}

func (s *PGUserStore) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, id)
	return err
}

func (s *PGUserStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM users WHERE id = $1`, id)
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
