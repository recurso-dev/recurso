-- Beat Lago A3: pay-in-advance charges.
--
-- A pay_in_advance charge is rated per usage event at ingestion time and
-- captured immediately as a pending unbilled_charge (which GenerateInvoice
-- folds onto the subscription's next invoice), rather than aggregated at
-- period close. Only non-cumulative charge models (per_unit, percentage,
-- dynamic) may be pay-in-advance — the tier/bundle math of graduated/volume/
-- graduated_percentage/package is period-cumulative and has no per-event tier.
--
-- Existing charges default to FALSE (arrears), so nothing changes for them.

ALTER TABLE plan_charges
    ADD COLUMN IF NOT EXISTS pay_in_advance BOOLEAN NOT NULL DEFAULT FALSE;
