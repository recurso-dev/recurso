-- Add missing next_retry_at column to event_deliveries
ALTER TABLE event_deliveries ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
