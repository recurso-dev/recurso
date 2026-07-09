-- US sales-tax nexus (Phase 1: config + gating, no threshold data).
-- A tenant declares the US states it has nexus in; the tax resolver then
-- collects US sales tax only in those states (see resolveUSSalesTax). This is
-- pure configuration — it encodes no state tax rule that could be wrong, only
-- what the tenant declares. Economic-nexus threshold tracking is Phase 2.
CREATE TABLE tenant_tax_nexus (
    id             UUID PRIMARY KEY,
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    state_code     CHAR(2) NOT NULL,
    nexus_type     TEXT NOT NULL DEFAULT 'physical'
                   CHECK (nexus_type IN ('physical', 'voluntary', 'economic')),
    established_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, state_code)
);

CREATE INDEX idx_tenant_tax_nexus_tenant_id ON tenant_tax_nexus(tenant_id);
