//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

func (s *SQLiteUserStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *SQLiteUserStore) ListUsers(ctx context.Context, offset, limit int) ([]store.User, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+userSelectCols+` FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
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
