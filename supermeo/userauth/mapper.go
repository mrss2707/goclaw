package userauth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type UserTenantMapper struct {
	tenants store.TenantStore
}

func NewUserTenantMapper(tenants store.TenantStore) *UserTenantMapper {
	return &UserTenantMapper{tenants: tenants}
}

func (m *UserTenantMapper) Tenants() store.TenantStore {
	return m.tenants
}

func (m *UserTenantMapper) GetOrCreateTenant(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	uidStr := userID.String()

	// Nếu user đã là member của Master tenant → dùng Master.
	if role, err := m.tenants.GetUserRole(ctx, store.MasterTenantID, uidStr); err == nil && role != "" {
		return store.MasterTenantID, nil
	}

	// Nếu Master tenant chưa có ai → đây là user đầu tiên → tự động thêm vào Master.
	if users, err := m.tenants.ListUsers(ctx, store.MasterTenantID); err == nil && len(users) == 0 {
		if addErr := m.tenants.AddUser(ctx, store.MasterTenantID, uidStr, "owner"); addErr != nil {
			if !store.IsDuplicateKeyErr(addErr) {
				slog.Warn("userauth.mapper.add_to_master_failed", "user_id", userID, "error", addErr)
			}
		}
		slog.Info("userauth.mapper.first_user_added_to_master", "user_id", userID)
		return store.MasterTenantID, nil
	}

	// --- existing personal-tenant logic (UNCHANGED below) ---
	slug := "user-" + userID.String()

	tenant, err := m.tenants.GetTenantBySlug(ctx, slug)
	if err == nil {
		return tenant.ID, nil
	}

	t := &store.TenantData{
		Name:   slug,
		Slug:   slug,
		Status: "active",
	}
	if createErr := m.tenants.CreateTenant(ctx, t); createErr != nil {
		if dupErr := store.IsDuplicateKeyErr(createErr); dupErr {
			tenant, getErr := m.tenants.GetTenantBySlug(ctx, slug)
			if getErr != nil {
				return uuid.Nil, fmt.Errorf("userauth: tenant race: %w", getErr)
			}
			tenantID := tenant.ID
			if addErr := m.tenants.AddUser(ctx, tenantID, uidStr, "owner"); addErr != nil {
				if !store.IsDuplicateKeyErr(addErr) {
					slog.Warn("userauth.mapper.add_user_failed", "user_id", userID, "tenant_id", tenantID, "error", addErr)
				}
			}
			return tenantID, nil
		}
		return uuid.Nil, fmt.Errorf("userauth: create tenant: %w", createErr)
	}

	tenantID := t.ID
	if addErr := m.tenants.AddUser(ctx, tenantID, uidStr, "owner"); addErr != nil {
		if !store.IsDuplicateKeyErr(addErr) {
			slog.Warn("userauth.mapper.add_user_failed", "user_id", userID, "tenant_id", tenantID, "error", addErr)
		}
	}

	slog.Info("userauth.mapper.created_tenant", "user_id", userID, "tenant_id", tenantID, "slug", slug)
	return tenantID, nil
}
