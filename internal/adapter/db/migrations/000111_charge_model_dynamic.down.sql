-- Revert `dynamic`: restore the 6-model CHECK (000110 state) and drop the
-- per-event column. Fails if any plan_charges row still uses 'dynamic'.

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package', 'percentage', 'graduated_percentage'));

ALTER TABLE usage_events DROP COLUMN IF EXISTS dynamic_amount;
