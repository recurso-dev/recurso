-- The raw-event stream (GET /v1/usage/events) orders the tenant's events
-- newest-first through the subscriptions join. Give the planner a global
-- newest-first path so the unfiltered default view stays cheap as the
-- table grows; per-subscription/customer filters keep their own indexes.
CREATE INDEX IF NOT EXISTS idx_usage_events_time ON usage_events(timestamp DESC);
