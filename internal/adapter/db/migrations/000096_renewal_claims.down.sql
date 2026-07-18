DROP INDEX IF EXISTS idx_subscriptions_renewal_due;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS renewal_claimed_at;
