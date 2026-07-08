-- The cancel/pause flow writes cancellation_reason and cancellation_feedback on
-- the subscriptions table (see subscription_repository UpdateStatus/cancel and
-- domain.Subscription), but only cancel_at_period_end (000024) and canceled_at
-- were ever added — these two columns were never created, so cancel/pause failed
-- with "column cancellation_reason of relation subscriptions does not exist".
-- NOT NULL DEFAULT '' matches the non-pointer string fields on the domain struct.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS cancellation_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS cancellation_feedback TEXT NOT NULL DEFAULT '';
