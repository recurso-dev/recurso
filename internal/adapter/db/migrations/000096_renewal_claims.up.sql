-- Lago-parity A1 (spec_lago_parity.md): the billing-cycle scheduler's
-- at-most-once claim. renewal_claimed_at is a lease: a claimed subscription
-- is invisible to other runners until the lease expires, and a successful
-- renewal advances current_period_end so the row stops being due at all.
-- Failure = lease expiry = automatic retry on a later tick.

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS renewal_claimed_at TIMESTAMPTZ;

-- Due-scan hot path: active subscriptions ordered by period end.
CREATE INDEX IF NOT EXISTS idx_subscriptions_renewal_due
    ON subscriptions (current_period_end)
    WHERE status = 'active';
