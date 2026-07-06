-- Dunning recovery attribution: one row per invoice that transitioned to paid
-- after at least one failed payment attempt (retry_count >= 1 or an active
-- dunning action/campaign). Written at the moment of payment; the UNIQUE
-- constraint on invoice_id makes recording idempotent across the multiple
-- payment-success paths (webhooks, retry worker, offline reconciliation).
CREATE TABLE recovered_payments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    invoice_id UUID NOT NULL UNIQUE,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    strategy TEXT NOT NULL DEFAULT '',
    campaign_id UUID,
    days_to_recover INT NOT NULL DEFAULT 0,
    recovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovered_payments_tenant_time ON recovered_payments(tenant_id, recovered_at);
