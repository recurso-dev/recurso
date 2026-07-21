DROP TABLE IF EXISTS progressive_billing_watermarks;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS progressive_billing_threshold;
