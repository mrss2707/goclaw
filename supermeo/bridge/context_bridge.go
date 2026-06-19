package bridge

import (
	"context"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

func WithUserUUID(ctx context.Context, id uuid.UUID) context.Context {
	return store.WithUserUUID(ctx, id)
}

func UserUUIDFromContext(ctx context.Context) uuid.UUID {
	return store.UserUUIDFromContext(ctx)
}

func WithTenantID(ctx context.Context, id uuid.UUID) context.Context {
	return store.WithTenantID(ctx, id)
}

func TenantIDFromContext(ctx context.Context) uuid.UUID {
	return store.TenantIDFromContext(ctx)
}

func WithRole(ctx context.Context, role string) context.Context {
	return store.WithRole(ctx, role)
}

func WithLocale(ctx context.Context, locale string) context.Context {
	return store.WithLocale(ctx, locale)
}

func RoleFromContext(ctx context.Context) string {
	return store.RoleFromContext(ctx)
}
