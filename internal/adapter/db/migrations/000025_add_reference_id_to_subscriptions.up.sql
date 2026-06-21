ALTER TABLE subscriptions ADD COLUMN reference_id VARCHAR(100);
CREATE INDEX idx_subscriptions_reference_id ON subscriptions(reference_id);
