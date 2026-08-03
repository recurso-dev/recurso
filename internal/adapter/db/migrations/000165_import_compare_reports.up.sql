-- Persisted migration Compare runs — each one a citable receipt that a
-- migration was proven (coverage / fidelity / continuity) before cut-over.
CREATE TABLE IF NOT EXISTS import_compare_reports (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    ready BOOLEAN NOT NULL,
    report JSONB NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_import_compare_reports_tenant
    ON import_compare_reports (tenant_id, generated_at DESC);
