package bridge

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type ProviderResolver struct {
	users        store.UserStore
	userProviderReg store.UserProviderStore
}

func NewProviderResolver(users store.UserStore, upReg store.UserProviderStore) *ProviderResolver {
	return &ProviderResolver{users: users, userProviderReg: upReg}
}

func (r *ProviderResolver) ResolveForAgent(ctx context.Context, userID uuid.UUID, agentProvider string) (providerName string, apiKey string, err error) {
	if r.userProviderReg == nil {
		return "", "", nil
	}
	up, err := r.userProviderReg.GetProviderByName(ctx, userID, agentProvider)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if !up.Enabled {
		return "", "", nil
	}
	return up.Name, up.APIKey, nil
}
