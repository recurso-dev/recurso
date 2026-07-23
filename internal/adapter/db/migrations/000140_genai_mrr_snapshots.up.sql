-- Expose the daily MRR snapshot history to the GenAI "Ask your data" schema so
-- users can ask MRR growth / churn-trend / historical-metric questions. Same
-- hardening pattern as 000077: a tenant-scoped view over public.mrr_snapshots
-- (scoping baked in via the app.tenant_id GUC), readable ONLY by the
-- genai_readonly role. One row per (subscription, snapshot_date); mrr_amount is
-- in the currency's minor units. Exposes no tenant_id column — same as the other
-- genai views.
CREATE OR REPLACE VIEW genai.mrr_snapshots AS
    SELECT subscription_id, customer_id, plan_id, snapshot_date, mrr_amount, created_at
    FROM public.mrr_snapshots
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;

GRANT SELECT ON genai.mrr_snapshots TO genai_readonly;
