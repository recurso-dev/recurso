-- Beat Lago A1.1: add the `percentage` charge model (charge models 4 -> 7).
--
-- plan_charges.charge_model is guarded by a CHECK constraint enumerating the
-- supported models. Widen it to admit 'percentage'. The percentage model
-- prices a percentage of the aggregated monetary base (a sum in minor units),
-- plus an optional flat fee, with an optional free-units allowance and
-- optional min/max clamps — all carried in the existing JSONB `amounts`
-- column, so no column change is needed here.
--
-- Existing rows are unaffected: they already satisfy the widened constraint.

ALTER TABLE plan_charges DROP CONSTRAINT IF EXISTS plan_charges_charge_model_check;

ALTER TABLE plan_charges
    ADD CONSTRAINT plan_charges_charge_model_check
    CHECK (charge_model IN ('per_unit', 'graduated', 'volume', 'package', 'percentage'));
