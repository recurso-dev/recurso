-- The webhook delivery worker's claim loop runs continuously:
--   WHERE delivered_at IS NULL AND (next_retry_at IS NULL OR next_retry_at <= NOW())
--   ORDER BY created_at LIMIT $n FOR UPDATE SKIP LOCKED
-- Neither existing index (event_id, endpoint_id) serves it, so every tick
-- scanned the WHOLE deliveries table, growing with all-time delivery history.
-- A partial index over only-undelivered rows stays tiny regardless of history
-- and serves both the filter and the created_at ordering.
CREATE INDEX IF NOT EXISTS idx_event_deliveries_undelivered
    ON event_deliveries (created_at)
    WHERE delivered_at IS NULL;
