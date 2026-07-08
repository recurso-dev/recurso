-- Per-product HSN codes (Phase 2: catalog HSN + per-line rates).
-- Each plan (base or add-on) carries its own HSN/SAC code so an invoice line is
-- taxed at that plan's rate instead of always the tenant SAC.
-- An empty hsn_code preserves Phase-1 behaviour exactly (falls back to the
-- tenant SAC, then 998314, at tax-resolution time).
ALTER TABLE plans ADD COLUMN IF NOT EXISTS hsn_code TEXT NOT NULL DEFAULT '';
