-- Per-entity MRR: stamp each MRR snapshot with its legal entity so MRR/growth/
-- churn can be reported per entity and consolidated (Multi-Entity Books follow-up).
--
-- Unlike the transactional money-path tables (which keep primary = NULL to stay
-- byte-identical), this is a DENORMALIZED REPORTING table: we store the CONCRETE
-- entity id (the subscription's entity, or the tenant's primary entity when the
-- subscription is on the primary/NULL). That makes per-entity filtering a plain
-- equality (entity_id = $x); a consolidated report is simply "no entity filter".
ALTER TABLE mrr_snapshots ADD COLUMN entity_id UUID;

-- Backfill: resolve each snapshot to its subscription's entity, defaulting to the
-- tenant's primary entity when the subscription is on the primary (entity_id NULL).
UPDATE mrr_snapshots m
SET entity_id = COALESCE(
    s.entity_id,
    (SELECT e.id FROM entities e WHERE e.tenant_id = m.tenant_id AND e.is_primary LIMIT 1)
)
FROM subscriptions s
WHERE m.subscription_id = s.id;

-- Any orphan snapshot whose subscription no longer exists → the tenant's primary.
UPDATE mrr_snapshots m
SET entity_id = (SELECT e.id FROM entities e WHERE e.tenant_id = m.tenant_id AND e.is_primary LIMIT 1)
WHERE m.entity_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_mrr_snapshots_entity
    ON mrr_snapshots (tenant_id, entity_id, snapshot_date);

-- GenAI "Ask your data": expose entity_id on the MRR view and add a tenant-scoped
-- entities view so the model can resolve an entity NAME to its id and answer
-- "What is <Entity>'s MRR growth?" (same hardening pattern as 000077/000140).
-- DROP+CREATE (not CREATE OR REPLACE): inserting entity_id mid-list changes an
-- existing column position, which CREATE OR REPLACE VIEW rejects.
DROP VIEW IF EXISTS genai.mrr_snapshots;
CREATE VIEW genai.mrr_snapshots AS
    SELECT subscription_id, customer_id, plan_id, entity_id, snapshot_date, mrr_amount, created_at
    FROM public.mrr_snapshots
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;
GRANT SELECT ON genai.mrr_snapshots TO genai_readonly;

CREATE OR REPLACE VIEW genai.entities AS
    SELECT id, name, legal_name, is_primary, country_code, created_at
    FROM public.entities
    WHERE tenant_id = current_setting('app.tenant_id', true)::uuid;
GRANT SELECT ON genai.entities TO genai_readonly;
