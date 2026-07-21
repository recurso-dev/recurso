-- Progressive billing (A5): interim invoices when accrued usage crosses a
-- threshold, reconciled against the double-billing guard via a per-period
-- billed-amount watermark.
--
-- A subscription with progressive_billing_threshold set bills its metered
-- charges incrementally: whenever the total unbilled usage amount reaches the
-- threshold, an interim invoice bills the delta since the last bill. The
-- watermark records how much has already been invoiced per (subscription,
-- charge, period), so each bill is exactly rate(cumulative_now) - billed_amount
-- and no usage unit is billed twice. NULL threshold = off (classic arrears).

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS progressive_billing_threshold BIGINT;

CREATE TABLE IF NOT EXISTS progressive_billing_watermarks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    charge_id UUID NOT NULL REFERENCES plan_charges(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    -- billed_amount is the total already invoiced for this charge in this period
    -- (minor units); the next bill is rate(cumulative_now) - billed_amount.
    billed_amount BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, charge_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_progressive_watermarks_sub_period
    ON progressive_billing_watermarks (subscription_id, period_start);
