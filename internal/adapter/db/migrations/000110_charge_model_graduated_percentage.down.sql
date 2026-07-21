-- Revert `graduated_percentage`: restore the 5-model CHECK (000109 state).
-- Fails if any plan_charges row still uses 'graduated_percentage'.

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package', 'percentage'));
