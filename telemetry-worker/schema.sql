-- Recurso telemetry receiver schema (D1 / SQLite).
-- One row per event; instance_id is the client's random anonymous UUID.
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  received_at TEXT NOT NULL, -- server receive time, RFC3339 UTC
  event TEXT NOT NULL,
  instance_id TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  client_ts TEXT NOT NULL DEFAULT '',
  props TEXT NOT NULL DEFAULT '{}' -- coarse documented props only (deployment, buckets, ...)
);
CREATE INDEX IF NOT EXISTS idx_events_instance ON events (instance_id);
CREATE INDEX IF NOT EXISTS idx_events_event ON events (event);
CREATE INDEX IF NOT EXISTS idx_events_received ON events (received_at);
