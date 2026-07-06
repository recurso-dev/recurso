-- Support time-windowed usage reads:
--   GET /v1/usage                     (subscription_id/customer_id + timestamp range)
--   GET /v1/subscriptions/{id}/usage  (per-period aggregation)
CREATE INDEX IF NOT EXISTS idx_usage_sub_time ON usage_events(subscription_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_customer_time ON usage_events(customer_id, timestamp);
