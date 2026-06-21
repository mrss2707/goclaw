package bridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/gateway"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type DefaultUserAuthenticator struct {
	users  store.UserStore
	jwt    *userauth.JWTService
	mapper *userauth.UserTenantMapper
}

func NewDefaultUserAuthenticator(users store.UserStore, jwt *userauth.JWTService, mapper *userauth.UserTenantMapper) *DefaultUserAuthenticator {
	return &DefaultUserAuthenticator{
		users:  users,
		jwt:    jwt,
		mapper: mapper,
	}
}

func (a *DefaultUserAuthenticator) Authenticate(ctx context.Context, token string) (*gateway.UserAuthIdentity, error) {
	claims, err := a.jwt.Verify(token)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	user, err := a.users.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("authenticate: user not found")
		}
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	tenantID, err := a.mapper.GetOrCreateTenant(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	role := "admin"
	if dbRole, err := a.mapper.Tenants().GetUserRole(ctx, tenantID, user.ID.String()); err == nil && dbRole != "" {
		role = dbRole
	}

	return &gateway.UserAuthIdentity{
		UserID:   user.ID,
		TenantID: tenantID,
		Role:     role,
		Locale:   user.Locale,
		Email:    derefStr(user.Email),
	}, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
