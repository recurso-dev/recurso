-- Per-tenant payment-gateway credentials (BYO gateway). Secret columns hold
-- AES-256-GCM ciphertext (see internal/adapter/secretbox), never plaintext.
-- At most one active connection per (tenant, provider) — enforced by a partial
-- unique index so disconnected (inactive) rows can coexist for the audit trail.
CREATE TABLE IF NOT EXISTS gateway_connections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL CHECK (provider IN ('stripe', 'razorpay')),
    mode                TEXT NOT NULL DEFAULT 'test' CHECK (mode IN ('test', 'live')),
    public_key          TEXT NOT NULL DEFAULT '',
    secret_key_enc      TEXT NOT NULL DEFAULT '',
    webhook_secret_enc  TEXT NOT NULL DEFAULT '',
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_gateway_connections_active
    ON gateway_connections (tenant_id, provider)
    WHERE active;

-- Webhook routing resolves a connection by id from the per-connection URL
-- (/webhooks/stripe/:connID); the PK already covers that lookup.
CREATE INDEX IF NOT EXISTS ix_gateway_connections_tenant
    ON gateway_connections (tenant_id);
