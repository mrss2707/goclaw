package pg

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

func (s *PGUserStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *PGUserStore) ListUsers(ctx context.Context, offset, limit int) ([]store.User, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, google_id, password_hash, display_name, avatar_url, email_verified, verification_code, verification_exp, locale, metadata, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []store.User
	for rows.Next() {
		var u store.User
		if err := rows.Scan(&u.ID, &u.Email, &u.GoogleID, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
			&u.EmailVerified, &u.VerificationCode, &u.VerificationExp, &u.Locale, &u.Metadata, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return []store.User{}, total, nil
	}
	return users, total, nil
}
