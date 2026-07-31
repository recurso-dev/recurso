-- Track how many billing periods a subscription's coupon has been applied to,
-- so a `repeating` (N-month) coupon can apply for its first N periods and then
-- stop. `forever` applies every period and `once` only the first regardless of
-- this counter; it exists for `repeating`. (coupon_id itself already exists from
-- migration 000006 — it was simply never loaded/persisted by the repo until now.)
ALTER TABLE subscriptions ADD COLUMN coupon_periods_applied INT NOT NULL DEFAULT 0;
