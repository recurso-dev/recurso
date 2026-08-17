-- Recurso Cloud self-billing ("Recurso runs on Recurso"), Increment 2.
--
-- The usage meter. For each tenant and billing period, this records what the
-- tenant did on Recurso — the numbers the founder will charge on later:
--   * tracked_revenue_minor  = everything the tenant invoiced (paid or not)
--   * collected_volume_minor = payments the tenant actually collected
-- broken out per currency so no FX conversion is needed to store a reading.
--
-- This increment is money-free: rows here are measurements only. Turning a
-- reading into a charge (the Recurso Cloud plan + subscription + invoice) is a
-- later increment, gated by the invariant harness. The UNIQUE key makes the
-- daily re-measurement of the current period an idempotent upsert.
CREATE TABLE IF NOT EXISTS cloud_tenant_usage (
    id                     UUID PRIMARY KEY,
    tenant_id              UUID NOT NULL,
    period_start           TIMESTAMPTZ NOT NULL,
    period_end             TIMESTAMPTZ NOT NULL,
    currency               VARCHAR(10) NOT NULL,
    tracked_revenue_minor  BIGINT NOT NULL DEFAULT 0,
    collected_volume_minor BIGINT NOT NULL DEFAULT 0,
    computed_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, period_start, currency)
);

CREATE INDEX IF NOT EXISTS idx_cloud_tenant_usage_tenant
    ON cloud_tenant_usage (tenant_id, period_start);
