DROP INDEX IF EXISTS idx_subscriptions_resume_at;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS resume_at;
