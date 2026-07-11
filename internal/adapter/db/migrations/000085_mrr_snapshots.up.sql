-- Per-subscription MRR history, one row per active subscription per snapshot
-- date. The MRR waterfall (new/expansion/contraction/churned/reactivation) is
-- computed by diffing two snapshot dates per subscription; there is no other
-- source of period-over-period movement (MRR is otherwise only ever computed
-- live). Amounts are monthly-normalized minor units in the subscription's own
-- currency; the query layer converts to a reporting currency.
CREATE TABLE IF NOT EXISTS mrr_snapshots (
    tenant_id       UUID        NOT NULL,
    subscription_id UUID        NOT NULL,
    snapshot_date   DATE        NOT NULL,
    mrr_amount      BIGINT      NOT NULL,
    currency        VARCHAR(3)  NOT NULL,
    customer_id     UUID,
    plan_id         UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, subscription_id, snapshot_date)
);

-- Loading a whole tenant's snapshot on a given date is the hot path.
CREATE INDEX IF NOT EXISTS idx_mrr_snapshots_tenant_date
    ON mrr_snapshots (tenant_id, snapshot_date);
