-- Hot-path indexes for rev-rec queries that had no candidate index (Postgres
-- does not auto-index FK columns), forcing sequential scans that grow with
-- TOTAL data volume across all tenants:
--
--   1. SumPendingRecognitionEvents (the deferred>=scheduled reconciler
--      invariant) filters tenant_id + status='pending' and runs on EVERY
--      reconciliation — the finance page, the E2E gate, and every invariant-
--      harness step.
--   2. GetActiveScheduleByInvoice filters invoice_id + status='active' and
--      runs on EVERY invoice payment (the schedule-idempotency check in
--      MarkInvoicePaid) and every refund unwind.
--   3. GetActiveSchedulesBySubscription filters subscription_id and runs on
--      every plan change, cancel unwind, and downgrade reversal.
CREATE INDEX IF NOT EXISTS idx_revrec_events_tenant_pending
    ON recognition_events (tenant_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_revrec_schedules_invoice
    ON revenue_schedules (invoice_id);
CREATE INDEX IF NOT EXISTS idx_revrec_schedules_subscription
    ON revenue_schedules (subscription_id);
