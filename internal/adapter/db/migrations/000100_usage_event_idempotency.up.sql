-- Ingestion hardening (Lago-parity C1).
--
-- transaction_id is the caller's idempotency key: a duplicate
-- (subscription_id, transaction_id) collapses to the original event, so
-- retry-happy SDK ingestion can never inflate usage. NULL for callers that
-- don't send one (unconstrained, as before).
ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS uq_usage_events_sub_txid
    ON usage_events (subscription_id, transaction_id)
    WHERE transaction_id IS NOT NULL;

-- Covering index for the rating/aggregation hot path:
-- WHERE subscription_id = ? AND dimension = ? AND timestamp >= ? AND < ?.
CREATE INDEX IF NOT EXISTS idx_usage_events_sub_dim_ts
    ON usage_events (subscription_id, dimension, timestamp);
