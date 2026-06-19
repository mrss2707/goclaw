package pg

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/internal/store/base"
)

type PGUserSessionStore struct {
	db *sql.DB
}

func NewPGUserSessionStore(db *sql.DB) *PGUserSessionStore {
	return &PGUserSessionStore{db: db}
}

func (s *PGUserSessionStore) CreateSession(ctx context.Context, session *store.UserSession) error {
	now := time.Now()
	session.ID = store.GenNewID()
	session.CreatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_sessions (id, user_id, jti, expires_at, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		session.ID, session.UserID, session.JTI, session.ExpiresAt,
		base.JsonOrEmpty(session.Metadata), session.CreatedAt,
	)
	if err != nil && isDuplicateKeyErr(err) {
		return store.ErrDuplicateKey
	}
	return err
}

func (s *PGUserSessionStore) GetSessionByJTI(ctx context.Context, jti string) (*store.UserSession, error) {
	var sess store.UserSession
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, jti, expires_at, metadata, created_at
		 FROM user_sessions WHERE jti = $1`, jti,
	).Scan(&sess.ID, &sess.UserID, &sess.JTI, &sess.ExpiresAt, &sess.Metadata, &sess.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *PGUserSessionStore) DeleteSession(ctx context.Context, jti string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM user_sessions WHERE jti = $1`, jti)
	return err
}

func (s *PGUserSessionStore) DeleteAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM user_sessions WHERE user_id = $1`, userID)
	return err
}

func (s *PGUserSessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM user_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
