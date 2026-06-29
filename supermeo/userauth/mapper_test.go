package userauth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// configurableTenantStore implements store.TenantStore with per-test control
// over return values. Each method can be overridden by setting the corresponding
// field. Nil fields use sensible defaults.
type configurableTenantStore struct {
	// GetUserRole returns (role, nil) when role != ""; ("", nil) when role == ""; (role, err) when err != nil.
	getUserRoleRole string
	getUserRoleErr  error

	// ListUsers returns (users, nil) when err == nil; (nil, err) otherwise.
	listUsers     []store.TenantUserData
	listUsersErr  error

	// GetTenantBySlug returns (tenant, nil) when found; (nil, err) when not.
	getTenantBySlugTenant *store.TenantData
	getTenantBySlugErr    error

	// CreateTenant sets the tenant ID on the passed *TenantData when called.
	createTenantID  uuid.UUID
	createTenantErr error

	// AddUser records calls.
	addUserCalls     []addUserCall
	addUserErr       error
}

type addUserCall struct {
	tenantID uuid.UUID
	userID   string
	role     string
}

func (m *configurableTenantStore) CreateTenant(_ context.Context, t *store.TenantData) error {
	if m.createTenantErr != nil {
		return m.createTenantErr
	}
	t.ID = m.createTenantID
	return nil
}

func (m *configurableTenantStore) GetTenant(_ context.Context, id uuid.UUID) (*store.TenantData, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *configurableTenantStore) GetTenantBySlug(_ context.Context, slug string) (*store.TenantData, error) {
	if m.getTenantBySlugErr != nil {
		return nil, m.getTenantBySlugErr
	}
	return m.getTenantBySlugTenant, nil
}

func (m *configurableTenantStore) ListTenants(context.Context) ([]store.TenantData, error) { return nil, nil }
func (m *configurableTenantStore) UpdateTenant(context.Context, uuid.UUID, map[string]any) error {
	return nil
}

func (m *configurableTenantStore) AddUser(_ context.Context, tenantID uuid.UUID, userID, role string) error {
	m.addUserCalls = append(m.addUserCalls, addUserCall{tenantID, userID, role})
	return m.addUserErr
}

func (m *configurableTenantStore) RemoveUser(context.Context, uuid.UUID, string) error { return nil }

func (m *configurableTenantStore) GetUserRole(_ context.Context, tenantID uuid.UUID, userID string) (string, error) {
	if m.getUserRoleErr != nil {
		return "", m.getUserRoleErr
	}
	return m.getUserRoleRole, nil
}

func (m *configurableTenantStore) ListUsers(_ context.Context, tenantID uuid.UUID) ([]store.TenantUserData, error) {
	if m.listUsersErr != nil {
		return nil, m.listUsersErr
	}
	return m.listUsers, nil
}

func (m *configurableTenantStore) ListUserTenants(context.Context, string) ([]store.TenantUserData, error) {
	return nil, nil
}

func (m *configurableTenantStore) ResolveUserTenant(context.Context, string) (uuid.UUID, error) {
	return store.MasterTenantID, nil
}

func (m *configurableTenantStore) GetTenantUser(context.Context, uuid.UUID) (*store.TenantUserData, error) {
	return nil, errors.New("not found")
}

func (m *configurableTenantStore) CreateTenantUserReturning(context.Context, uuid.UUID, string, string, string) (*store.TenantUserData, error) {
	return nil, errors.New("not implemented")
}

func (m *configurableTenantStore) GetTenantsByIDs(context.Context, []uuid.UUID) ([]store.TenantData, error) {
	return nil, nil
}

var _ store.TenantStore = (*configurableTenantStore)(nil)

func TestGetOrCreateTenant_UserInMaster(t *testing.T) {
	st := &configurableTenantStore{
		getUserRoleRole: "owner",
	}
	mapper := NewUserTenantMapper(st)

	userID := uuid.MustParse("019ef56d-845a-7a38-a235-f223011a6629")
	tenantID, err := mapper.GetOrCreateTenant(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != store.MasterTenantID {
		t.Errorf("expected MasterTenantID %s, got %s", store.MasterTenantID, tenantID)
	}
	if len(st.addUserCalls) > 0 {
		t.Errorf("expected no AddUser calls, got %d", len(st.addUserCalls))
	}
}

func TestGetOrCreateTenant_FirstUserAutoAddedToMaster(t *testing.T) {
	st := &configurableTenantStore{
		getUserRoleRole: "",    // not a member
		listUsers:       []store.TenantUserData{}, // Master has no users
	}
	mapper := NewUserTenantMapper(st)

	userID := uuid.MustParse("019ef56d-845a-7a38-a235-f223011a6629")
	tenantID, err := mapper.GetOrCreateTenant(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != store.MasterTenantID {
		t.Errorf("expected MasterTenantID %s, got %s", store.MasterTenantID, tenantID)
	}
	if len(st.addUserCalls) != 1 {
		t.Fatalf("expected 1 AddUser call, got %d", len(st.addUserCalls))
	}
	call := st.addUserCalls[0]
	if call.tenantID != store.MasterTenantID {
		t.Errorf("AddUser tenant: expected %s, got %s", store.MasterTenantID, call.tenantID)
	}
	if call.userID != userID.String() {
		t.Errorf("AddUser userID: expected %s, got %s", userID.String(), call.userID)
	}
	if call.role != "owner" {
		t.Errorf("AddUser role: expected owner, got %s", call.role)
	}
}

func TestGetOrCreateTenant_RegularUserGetsPersonalTenant(t *testing.T) {
	personalID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	st := &configurableTenantStore{
		getUserRoleRole: "", // not in Master
		listUsers: []store.TenantUserData{
			{UserID: "some-other-user", Role: "owner"},
		},
		getTenantBySlugErr: errors.New("not found"),
		createTenantID:     personalID,
	}
	mapper := NewUserTenantMapper(st)

	userID := uuid.MustParse("019ef56d-845a-7a38-a235-f223011a6629")
	tenantID, err := mapper.GetOrCreateTenant(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID == store.MasterTenantID {
		t.Errorf("expected personal tenant, got MasterTenantID")
	}
	if tenantID != personalID {
		t.Errorf("expected %s, got %s", personalID, tenantID)
	}
}

func TestGetOrCreateTenant_ErrorsFallThrough(t *testing.T) {
	personalID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	st := &configurableTenantStore{
		getUserRoleErr:      errors.New("db connection refused"),
		listUsersErr:        errors.New("db connection refused"),
		getTenantBySlugErr:  errors.New("not found"),
		createTenantID:      personalID,
	}
	mapper := NewUserTenantMapper(st)

	userID := uuid.MustParse("019ef56d-845a-7a38-a235-f223011a6629")
	tenantID, err := mapper.GetOrCreateTenant(context.Background(), userID)

	if err != nil {
		t.Fatalf("expected no error (fall through), got: %v", err)
	}
	if tenantID != personalID {
		t.Errorf("expected personal tenant %s, got %s", personalID, tenantID)
	}
}

func TestGetOrCreateTenant_ExistingPersonalTenantReturned(t *testing.T) {
	personalID := uuid.MustParse("019ef56d-845d-79ad-b551-cf7a785c39b8")
	st := &configurableTenantStore{
		getUserRoleRole: "", // not in Master
		listUsers: []store.TenantUserData{
			{UserID: "existing-user", Role: "owner"},
		},
		getTenantBySlugTenant: &store.TenantData{
			ID:   personalID,
			Slug: "user-019ef56d-845a-7a38-a235-f223011a6629",
		},
	}
	mapper := NewUserTenantMapper(st)

	userID := uuid.MustParse("019ef56d-845a-7a38-a235-f223011a6629")
	tenantID, err := mapper.GetOrCreateTenant(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != personalID {
		t.Errorf("expected existing tenant %s, got %s", personalID, tenantID)
	}
}
