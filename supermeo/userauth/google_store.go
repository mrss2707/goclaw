package userauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type GoogleStore struct {
	users store.UserStore
}

func NewGoogleStore(users store.UserStore) *GoogleStore {
	return &GoogleStore{users: users}
}

func (s *GoogleStore) FindOrCreateUserByGoogle(ctx context.Context, info *GoogleUserInfo) (*store.User, error) {
	user, err := s.users.GetUserByGoogleID(ctx, info.GoogleID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup google_id: %w", err)
	}

	if info.Email != "" {
		user, err = s.users.GetUserByEmail(ctx, info.Email)
		if err == nil {
			if linkErr := s.users.LinkGoogleID(ctx, user.ID, info.GoogleID); linkErr != nil {
				if !store.IsDuplicateKeyErr(linkErr) {
					return nil, fmt.Errorf("link google_id: %w", linkErr)
				}
			}
			return user, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("lookup email: %w", err)
		}
	}

	u := &store.User{
		Email:         &info.Email,
		GoogleID:      &info.GoogleID,
		DisplayName:   &info.Name,
		AvatarURL:     &info.Picture,
		EmailVerified: true,
		Locale:        "en",
		Metadata:      []byte("{}"),
	}
	if err := s.users.CreateUser(ctx, u); err != nil {
		if store.IsDuplicateKeyErr(err) {
			if user, getErr := s.users.GetUserByEmail(ctx, info.Email); getErr == nil {
				return user, nil
			}
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
