DROP INDEX IF EXISTS idx_subscriptions_trialing_trial_end;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS trial_reminder_sent;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS trial_end;
