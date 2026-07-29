-- Import idempotency: maps a source system's object id (e.g. a Stripe
-- customer/price id) to the Recurso record it created, so re-running an import
-- skips what already landed instead of duplicating it.
CREATE TABLE IF NOT EXISTS import_external_refs (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source      TEXT NOT NULL,            -- 'stripe'
    kind        TEXT NOT NULL,            -- 'customer' | 'plan' | ...
    external_id TEXT NOT NULL,            -- the source system's id
    recurso_id  UUID NOT NULL,            -- the Recurso record created for it
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One Recurso record per (tenant, source, external id): the unique index is the
-- race-safe idempotency backstop even if two imports run concurrently.
CREATE UNIQUE INDEX IF NOT EXISTS import_external_refs_unique
    ON import_external_refs (tenant_id, source, external_id);
CREATE INDEX IF NOT EXISTS idx_import_external_refs_tenant_source
    ON import_external_refs (tenant_id, source);
