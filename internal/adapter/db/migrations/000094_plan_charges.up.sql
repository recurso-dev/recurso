-- Usage-based billing v1 (spec_usage_billing.md): plan charges + rating guard.
--
-- plan_charges attaches usage pricing for a billable metric to a plan.
-- amounts is per-currency pricing properties (JSONB keyed by ISO code):
--   per_unit:   {"USD": {"unit_amount": "0.0035"}}
--   package:    {"USD": {"package_amount": 500, "package_size": 1000}}
--   graduated/volume: {"USD": {"tiers": [{"up_to": 100, "unit_amount": "1"},
--                                        {"up_to": null, "unit_amount": "0.5"}]}}
-- unit_amount strings are decimal MAJOR currency units (sub-paise rates are
-- first-class, D1); flat/package amounts are int64 minor units.

CREATE TABLE IF NOT EXISTS plan_charges (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    metric_id UUID NOT NULL REFERENCES billable_metrics(id),
    charge_model TEXT NOT NULL CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package')),
    amounts JSONB NOT NULL DEFAULT '{}'::jsonb,
    hsn_code VARCHAR(20) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, metric_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_charges_tenant_plan
    ON plan_charges (tenant_id, plan_id);

-- usage_ratings is the double-billing guard: one row per (subscription,
-- charge, period_start) window ever rated onto an invoice. The UNIQUE
-- constraint makes rating idempotent — a retried invoice generation for an
-- already-rated window emits no metered lines (spec: rating idempotency).
CREATE TABLE IF NOT EXISTS usage_ratings (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    charge_id UUID NOT NULL REFERENCES plan_charges(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    invoice_id UUID NOT NULL,
    quantity BIGINT NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subscription_id, charge_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_ratings_invoice
    ON usage_ratings (invoice_id);
