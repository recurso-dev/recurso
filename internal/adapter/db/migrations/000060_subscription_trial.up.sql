-- Trial lifecycle support: a subscription may start in 'trialing' status with a
-- trial_end timestamp. A background scheduler converts trialing -> active at
-- trial_end (generating the first real invoice) and sends a trial-ending
-- reminder before expiry. trial_reminder_sent makes the reminder idempotent.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_end TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_reminder_sent BOOLEAN NOT NULL DEFAULT FALSE;

-- Partial index: the trial scheduler only ever scans trialing rows.
CREATE INDEX IF NOT EXISTS idx_subscriptions_trialing_trial_end
    ON subscriptions (trial_end)
    WHERE status = 'trialing';
