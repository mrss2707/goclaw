CREATE TABLE user_providers (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(50) NOT NULL,
    display_name    VARCHAR(255),
    provider_type   VARCHAR(30) NOT NULL DEFAULT 'openai_compat',
    api_base        TEXT,
    api_key         TEXT,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_user_providers_user ON user_providers(user_id);
