-- F2: recognition events are claimed (status -> 'processing') before their
-- ledger posting so concurrent workers get disjoint sets. claimed_at lets a
-- crashed worker's claims be requeued after a grace window.
ALTER TABLE recognition_events ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_revrec_events_processing
    ON recognition_events(claimed_at) WHERE status = 'processing';
