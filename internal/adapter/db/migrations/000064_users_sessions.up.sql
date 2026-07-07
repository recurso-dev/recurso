-- Admin-dashboard authentication: real user accounts + opaque sessions.
--
-- These sit alongside the existing tenant API-key auth (api_keys table). A
-- request is authenticated by EITHER a session cookie (a human logged into the
-- dashboard) OR a tenant API key (a machine / the demo). Both resolve to the
-- same tenant_id, so every existing tenant-scoped endpoint keeps working.

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,          -- always stored lower-cased by the app
    password_hash TEXT NOT NULL,          -- bcrypt
    name          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'member'
                    CHECK (role IN ('owner', 'admin', 'member')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-tenant email uniqueness (as specified) is guaranteed by the stricter
-- global unique index below, which additionally makes /auth/login (email +
-- password, no tenant supplied) unambiguous and race-free: one email == one
-- human account across the whole install.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_unique
    ON users (lower(email));

-- Tenant-scoped lookup/listing.
CREATE INDEX IF NOT EXISTS idx_users_tenant_email
    ON users (tenant_id, lower(email));

CREATE TABLE IF NOT EXISTS sessions (
    id         UUID PRIMARY KEY,
    token_hash TEXT NOT NULL,             -- SHA-256 (hex) of the opaque token; raw token never stored
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_agent TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS sessions_token_hash_unique ON sessions (token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);
