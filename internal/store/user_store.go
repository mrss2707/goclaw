package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrDuplicateKey = errors.New("duplicate key violates unique constraint")

func IsDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "23505")
}

// User represents a multi-user identity.
type User struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Email            *string    `json:"email" db:"email"`
	GoogleID         *string    `json:"google_id" db:"google_id"`
	PasswordHash     *string    `json:"-" db:"password_hash"`
	DisplayName      *string    `json:"display_name" db:"display_name"`
	AvatarURL        *string    `json:"avatar_url" db:"avatar_url"`
	EmailVerified    bool       `json:"email_verified" db:"email_verified"`
	VerificationCode *string    `json:"-" db:"verification_code"`
	VerificationExp  *time.Time `json:"-" db:"verification_exp"`
	Locale           string     `json:"locale" db:"locale"`
	Metadata         []byte     `json:"metadata" db:"metadata"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// UserSession is a JWT-tied session record for the session store blacklist.
type UserSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	JTI       string    `json:"jti" db:"jti"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Metadata  []byte    `json:"metadata" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserProvider represents a user-owned LLM provider configuration.
type UserProvider struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Name         string     `json:"name" db:"name"`
	DisplayName  *string    `json:"display_name" db:"display_name"`
	ProviderType string     `json:"provider_type" db:"provider_type"`
	APIBase      *string    `json:"api_base" db:"api_base"`
	APIKey       string     `json:"-" db:"api_key"`
	Enabled      bool       `json:"enabled" db:"enabled"`
	Settings     []byte     `json:"settings" db:"settings"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// UserTenantLink links a user to their tenant.
type UserTenantLink struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserStore manages multi-user identities.
type UserStore interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByGoogleID(ctx context.Context, googleID string) (*User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]any) error
	SetVerificationCode(ctx context.Context, id uuid.UUID, code string, exp time.Time) error
	VerifyEmail(ctx context.Context, id uuid.UUID) error
	LinkGoogleID(ctx context.Context, id uuid.UUID, googleID string) error
	UnlinkGoogleID(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// UserSessionStore manages JWT session tracking (blacklist).
type UserSessionStore interface {
	CreateSession(ctx context.Context, session *UserSession) error
	GetSessionByJTI(ctx context.Context, jti string) (*UserSession, error)
	DeleteSession(ctx context.Context, jti string) error
	DeleteAllUserSessions(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// UserProviderStore manages user-owned LLM providers.
type UserProviderStore interface {
	CreateProvider(ctx context.Context, p *UserProvider) error
	GetProvider(ctx context.Context, id uuid.UUID) (*UserProvider, error)
	GetProviderByName(ctx context.Context, userID uuid.UUID, name string) (*UserProvider, error)
	ListProviders(ctx context.Context, userID uuid.UUID) ([]UserProvider, error)
	UpdateProvider(ctx context.Context, id uuid.UUID, updates map[string]any) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error
}
