package admin

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type AdminUsersService struct {
	users store.UserStore
}

func NewAdminUsersService(users store.UserStore) *AdminUsersService {
	return &AdminUsersService{users: users}
}

func (s *AdminUsersService) ListUsers(ctx context.Context, search string, page, pageSize int) ([]store.User, int, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.users.ListUsers(ctx, offset, pageSize)
}

func (s *AdminUsersService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	slog.Info("admin.users.delete", "user_id", id)
	return s.users.DeleteUser(ctx, id)
}
