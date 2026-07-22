-- Scheduled auto-resume for paused subscriptions (issue #111). A retention
-- "pause" offer with pause_months, and any timed pause, records when the
-- subscription should return to active; a daily worker resumes elapsed ones.
-- NULL means an indefinite (manual-resume) pause — today's behaviour.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS resume_at TIMESTAMPTZ;

-- Partial index for the auto-resume claim: only paused subscriptions with a
-- scheduled resumption are ever scanned.
CREATE INDEX IF NOT EXISTS idx_subscriptions_resume_at
    ON subscriptions (resume_at)
    WHERE status = 'paused' AND resume_at IS NOT NULL;
