-- A1.3: add the `dynamic` charge model + per-event dynamic amount.
--
-- dynamic pricing lets the caller supply the exact price with each usage
-- event; a dynamic charge bills the sum of those amounts for the period.
--
-- 1) Carry the per-event price on usage_events (minor units, non-negative,
--    default 0 so existing rows and non-dynamic events are unaffected).
-- 2) Widen the plan_charges.charge_model CHECK to admit 'dynamic'.

ALTER TABLE usage_events
    ADD COLUMN IF NOT EXISTS dynamic_amount BIGINT NOT NULL DEFAULT 0;

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package', 'percentage', 'graduated_percentage', 'dynamic'));
