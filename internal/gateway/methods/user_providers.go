package methods

import (
	"context"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/gateway"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/pkg/protocol"
)

type UserProviderMethods struct {
	users    store.UserStore
	userProv store.UserProviderStore
}

func NewUserProviderMethods(users store.UserStore, up store.UserProviderStore) *UserProviderMethods {
	return &UserProviderMethods{users: users, userProv: up}
}

func (m *UserProviderMethods) Register(router *gateway.MethodRouter) {
	router.Register(protocol.MethodUserProvidersList, m.handleList)
	router.Register(protocol.MethodUserProvidersCreate, m.handleCreate)
	router.Register(protocol.MethodUserProvidersUpdate, m.handleUpdate)
	router.Register(protocol.MethodUserProvidersDelete, m.handleDelete)
}

func (m *UserProviderMethods) handleList(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	uid := store.UserUUIDFromContext(ctx)
	if uid == uuid.Nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrUnauthorized, "user auth required"))
		return
	}
	providers, err := m.userProv.ListProviders(ctx, uid)
	if err != nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInternal, err.Error()))
		return
	}
	client.SendResponse(protocol.NewOKResponse(req.ID, map[string]any{"providers": providers}))
}

func (m *UserProviderMethods) handleCreate(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotImplemented, "not yet implemented"))
}

func (m *UserProviderMethods) handleUpdate(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotImplemented, "not yet implemented"))
}

func (m *UserProviderMethods) handleDelete(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotImplemented, "not yet implemented"))
}
