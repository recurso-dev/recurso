DROP INDEX IF EXISTS idx_revrec_events_processing;
ALTER TABLE recognition_events DROP COLUMN IF EXISTS claimed_at;
