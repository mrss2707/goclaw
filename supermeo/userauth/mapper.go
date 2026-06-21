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
	slug := "user-" + userID.String()
	uidStr := userID.String()

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
