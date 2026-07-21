-- Beat Lago A1.2: add the `graduated_percentage` charge model.
--
-- Widen the plan_charges.charge_model CHECK to admit 'graduated_percentage'.
-- Its per-band percentage rates live in the existing JSONB `amounts` column
-- (each tier gains a `rate` field), so no column change is needed.
--
-- Existing rows are unaffected: they already satisfy the widened constraint.

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package', 'percentage', 'graduated_percentage'));
