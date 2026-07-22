DROP INDEX IF EXISTS idx_subscriptions_entity;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS entity_id;
