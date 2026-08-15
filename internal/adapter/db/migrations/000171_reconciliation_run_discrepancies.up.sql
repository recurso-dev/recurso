-- Per-run reconciliation discrepancy rows (lifecycle-history increment 2).
-- Migration 000168 records a run SUMMARY (counts) but discards the discrepancy
-- list, so a recorded run says "5 discrepancies" without saying WHICH 5 — the
-- historical run cannot be re-opened to explain itself. This persists the
-- already-computed discrepancy rows at record time so a recorded run becomes an
-- addressable, explainable object.
--
-- This does NOT change the reconciliation algorithm or the ephemeral live-run
-- GET path: it only stores the output that RecordRun already produced. The live
-- run caps its listed discrepancies (MaxListedDiscrepancies), so a run with more
-- than the cap stores the listed subset while total_discrepancies keeps the true
-- count. Runs recorded before this migration have no rows here (their detail was
-- never captured) — an honest empty list, not fabricated history.
CREATE TABLE IF NOT EXISTS reconciliation_run_discrepancies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id          UUID NOT NULL REFERENCES reconciliation_runs(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,
    type            TEXT NOT NULL,
    invoice_id      UUID,
    transaction_id  UUID,
    reference_id    UUID,
    account_code    INTEGER NOT NULL DEFAULT 0,
    expected_amount BIGINT NOT NULL DEFAULT 0,
    found_amount    BIGINT NOT NULL DEFAULT 0,
    seq             INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The per-run drill reads all rows for a run in insertion order.
CREATE INDEX IF NOT EXISTS idx_recon_run_discrepancies_run
    ON reconciliation_run_discrepancies (run_id, seq);
