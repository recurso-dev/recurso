DROP VIEW IF EXISTS genai.entities;

-- Restore the pre-000141 MRR view (without entity_id). DROP+CREATE because
-- dropping a mid-list column is not a valid CREATE OR REPLACE VIEW change.
DROP VIEW IF EXISTS genai.mrr_snapshots;
CREATE VIEW genai.mrr_snapshots AS
    SELECT subscription_id, customer_id, plan_id, snapshot_date, mrr_amount, created_at
    FROM public.mrr_snapshots
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;
GRANT SELECT ON genai.mrr_snapshots TO genai_readonly;

DROP INDEX IF EXISTS idx_mrr_snapshots_entity;
ALTER TABLE mrr_snapshots DROP COLUMN IF EXISTS entity_id;
