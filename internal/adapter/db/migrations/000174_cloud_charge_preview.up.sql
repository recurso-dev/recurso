-- Recurso Cloud self-billing ("Recurso runs on Recurso"), Increment 3 (dry-run).
--
-- What each tenant WOULD be charged for a period, in the reporting currency
-- (USD), computed from cloud_tenant_usage via the published pricing (free under
-- $10k tracked revenue, then the lower of 0.4% of collected volume or $99).
--
-- This is a PREVIEW only — no invoice, no ledger, no money. It exists so the
-- pricing + quota can be reviewed before real charging is turned on. The UNIQUE
-- key makes the daily recompute of the current period an idempotent upsert.
CREATE TABLE IF NOT EXISTS cloud_charge_preview (
    id                     UUID PRIMARY KEY,
    period_start           TIMESTAMPTZ NOT NULL,
    period_end             TIMESTAMPTZ NOT NULL,
    tenant_id              UUID NOT NULL,
    currency               VARCHAR(10) NOT NULL,
    tracked_revenue_minor  BIGINT NOT NULL DEFAULT 0,
    collected_volume_minor BIGINT NOT NULL DEFAULT 0,
    would_charge_minor     BIGINT NOT NULL DEFAULT 0,
    reason                 TEXT NOT NULL DEFAULT '',
    computed_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (period_start, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_cloud_charge_preview_period
    ON cloud_charge_preview (period_start);
