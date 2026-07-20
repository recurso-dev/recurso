-- Per-tenant credentials for operator-style integrations (tax providers, CRM,
-- storage) — the BYO analogue of gateway_connections for non-gateway services.
-- config_enc holds an AES-256-GCM sealed JSON blob of the provider's config
-- (see internal/adapter/secretbox), never plaintext. At most one active
-- connection per (tenant, category, provider).
CREATE TABLE IF NOT EXISTS integration_connections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category    TEXT NOT NULL CHECK (category IN ('tax', 'crm', 'storage')),
    provider    TEXT NOT NULL,
    config_enc  TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_integration_connections_active
    ON integration_connections (tenant_id, category, provider)
    WHERE active;

CREATE INDEX IF NOT EXISTS ix_integration_connections_tenant
    ON integration_connections (tenant_id);
