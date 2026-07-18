-- Usage-based billing v1 (spec_usage_billing.md): billable metrics.
--
-- A metric is a tenant-defined meter over usage_events. code doubles as the
-- event dimension it aggregates (VARCHAR(50) to match usage_events.dimension),
-- so existing events and the entitlement feature_key linkage keep working.
-- field_name is only set for the 'unique' aggregation: the event property
-- whose distinct values are counted.

CREATE TABLE IF NOT EXISTS billable_metrics (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    code VARCHAR(50) NOT NULL,
    aggregation_type TEXT NOT NULL CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique')),
    field_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

-- unique aggregation needs the property to count; the others must not set it.
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type = 'unique' AND field_name <> '') OR
        (aggregation_type <> 'unique' AND field_name = '')
    );
