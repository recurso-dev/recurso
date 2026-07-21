-- US sales-tax registrations (Track D · D4). Nexus tells you WHERE you must
-- collect; this records WHERE you're actually registered to do so, so the
-- dashboard can connect the dots — a state with nexus but no registration is a
-- compliance gap the tenant must close.

CREATE TABLE IF NOT EXISTS tax_registrations (
    id                  UUID PRIMARY KEY,
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    state_code          TEXT NOT NULL,
    registration_number TEXT NOT NULL DEFAULT '',
    -- registered: permit in hand; pending: applied, awaiting; not_registered:
    -- tracked but not yet registered (e.g. flagged from a nexus crossing).
    status              TEXT NOT NULL DEFAULT 'registered'
                          CHECK (status IN ('registered', 'pending', 'not_registered')),
    registered_at       DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, state_code)
);

CREATE INDEX IF NOT EXISTS idx_tax_registrations_tenant ON tax_registrations (tenant_id);
