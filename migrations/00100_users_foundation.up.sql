CREATE TABLE users (
    id                UUID PRIMARY KEY,
    email             VARCHAR(255),
    google_id         VARCHAR(255),
    password_hash     VARCHAR(255),
    display_name      VARCHAR(255),
    avatar_url        TEXT,
    email_verified    BOOLEAN NOT NULL DEFAULT false,
    verification_code VARCHAR(6),
    verification_exp  TIMESTAMPTZ,
    locale            VARCHAR(5) NOT NULL DEFAULT 'en',
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;

CREATE TABLE user_sessions (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    jti          VARCHAR(255) NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_jti ON user_sessions(jti);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);

CREATE TABLE user_tenant_links (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_user_tenant_links_user ON user_tenant_links(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tenant_links_tenant ON user_tenant_links(tenant_id);
