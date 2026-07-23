DROP INDEX IF EXISTS idx_revenue_schedules_entity;
ALTER TABLE revenue_schedules DROP COLUMN IF EXISTS entity_id;
