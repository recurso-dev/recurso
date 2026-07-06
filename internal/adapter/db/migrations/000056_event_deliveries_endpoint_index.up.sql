-- Support listing deliveries per webhook endpoint (GET /v1/webhooks/{id}/deliveries)
CREATE INDEX IF NOT EXISTS idx_event_deliveries_endpoint ON event_deliveries(webhook_endpoint_id);
