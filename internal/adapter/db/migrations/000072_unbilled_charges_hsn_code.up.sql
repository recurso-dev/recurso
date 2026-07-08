-- Per-product HSN codes & itemized invoice tax (Phase 3: charges as own lines).
-- A one-time / unbilled charge now carries its own HSN/SAC code so that, when it
-- is folded onto an invoice as its own line item, it is taxed at that code's
-- rate instead of the base plan's. An empty hsn_code preserves prior behaviour
-- exactly (falls back to the tenant SAC, then 998314, at tax-resolution time).
ALTER TABLE unbilled_charges ADD COLUMN IF NOT EXISTS hsn_code TEXT NOT NULL DEFAULT '';
