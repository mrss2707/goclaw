package admin

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/userauth"
)

type AdminSetupService struct {
	users      store.UserStore
	jwt        *userauth.JWTService
	mapper     *userauth.UserTenantMapper
	sysConfigs store.SystemConfigStore
}

func NewAdminSetupService(users store.UserStore, jwt *userauth.JWTService, mapper *userauth.UserTenantMapper, sysConfigs store.SystemConfigStore) *AdminSetupService {
	return &AdminSetupService{users: users, jwt: jwt, mapper: mapper, sysConfigs: sysConfigs}
}

func (s *AdminSetupService) IsSetupComplete(ctx context.Context) (bool, error) {
	count, err := s.users.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

type CreateAdminRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type CreateAdminResponse struct {
	Token    string         `json:"token"`
	UserID   uuid.UUID      `json:"user_id"`
	TenantID uuid.UUID      `json:"tenant_id"`
}

func (s *AdminSetupService) CreateFirstAdmin(ctx context.Context, req CreateAdminRequest) (*CreateAdminResponse, error) {
	complete, err := s.IsSetupComplete(ctx)
	if err != nil {
		return nil, err
	}
	if complete {
		return nil, errors.New("admin account already exists")
	}

	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}
	if err := userauth.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := userauth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = "Admin"
	}

	user := &store.User{
		Email:         &req.Email,
		PasswordHash:  &hash,
		DisplayName:   &displayName,
		EmailVerified: true,
		Locale:        "en",
		Metadata:      []byte("{}"),
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			return nil, errors.New("email already in use")
		}
		return nil, err
	}

	token, _, err := s.jwt.Sign(user.ID, req.Email)
	if err != nil {
		return nil, err
	}

	tenantID, err := s.mapper.GetOrCreateTenant(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if err := s.sysConfigs.Set(ctx, "supermeo.setup.complete", "true"); err != nil {
		slog.Warn("admin_setup.set_complete_flag", "error", err)
	}

	slog.Info("security.admin.first_admin_created", "user_id", user.ID, "email", maskEmail(req.Email))
	return &CreateAdminResponse{
		Token:    token,
		UserID:   user.ID,
		TenantID: tenantID,
	}, nil
}

func (s *AdminSetupService) CheckSetup(ctx context.Context) (bool, error) {
	complete, err := s.IsSetupComplete(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if complete {
		return true, nil
	}
	val, _ := s.sysConfigs.Get(ctx, "supermeo.setup.complete")
	return val == "true", nil
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) <= 2 {
		return email
	}
	return parts[0][:2] + "***@" + parts[1]
}
