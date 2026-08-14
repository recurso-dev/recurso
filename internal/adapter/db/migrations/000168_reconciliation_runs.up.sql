-- Reconciliation run history (lifecycle-history increment 1). Ledger
-- reconciliation is computed on demand and never persisted, so an operator can
-- see "is it balanced now?" but not "was it balanced last month, and who
-- checked?". This records a SUMMARY of each explicitly-recorded run — the audit
-- trail an auditor/controller needs — without changing the ephemeral GET path.
-- Only the counts are stored; the (large, per-run) discrepancy list stays
-- ephemeral. run_by is nullable (system/unauthenticated runs leave it null).
CREATE TABLE IF NOT EXISTS reconciliation_runs (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    run_by                 UUID,
    run_at                 TIMESTAMPTZ NOT NULL,
    invoices_checked       INTEGER NOT NULL DEFAULT 0,
    paid_invoices_checked  INTEGER NOT NULL DEFAULT 0,
    total_discrepancies    INTEGER NOT NULL DEFAULT 0,
    tb_compared            BOOLEAN NOT NULL DEFAULT FALSE,
    tb_accounts_checked    INTEGER NOT NULL DEFAULT 0,
    tb_transfers_checked   INTEGER NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_runs_tenant_created
    ON reconciliation_runs (tenant_id, created_at DESC);
