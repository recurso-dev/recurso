DROP INDEX IF EXISTS idx_usage_events_sub_dim_ts;
DROP INDEX IF EXISTS uq_usage_events_sub_txid;
ALTER TABLE usage_events DROP COLUMN IF EXISTS transaction_id;
