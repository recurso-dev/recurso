-- Usage threshold alerts (Lago-parity B3): fire once per billing period
-- per threshold when a subscription's aggregated usage crosses it.
-- last_fired_period_start is the dedup: the sweep claims it with a
-- conditional UPDATE before emitting, so concurrent sweeps fire once.

CREATE TABLE IF NOT EXISTS usage_alerts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    metric_code VARCHAR(50) NOT NULL,
    threshold_type TEXT NOT NULL CHECK (threshold_type IN ('quantity', 'percent_of_limit')),
    threshold BIGINT NOT NULL CHECK (threshold > 0),
    last_fired_period_start TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, metric_code, threshold_type, threshold)
);

CREATE INDEX IF NOT EXISTS idx_usage_alerts_tenant
    ON usage_alerts (tenant_id, subscription_id);
