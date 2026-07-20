-- Revert the `percentage` charge model: restore the original 4-model CHECK.
-- Fails if any plan_charges row still uses 'percentage' — remove those first.

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package'));
