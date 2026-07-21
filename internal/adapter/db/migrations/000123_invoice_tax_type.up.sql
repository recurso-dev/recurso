-- Persist the resolved tax type on the invoice (Track D · D3c). The invoice
-- already stores tax_amount, but a $0 tax line is ambiguous — exempt (a
-- certificate applied) vs no-nexus vs below-threshold. Recording the exact
-- evaluation reason returned by the tax engine makes the liability report's
-- exempt breakout unequivocal for audit.
--
-- Written once at invoice creation and never updated (no UPDATE touches it); the
-- liability report reads this column directly.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS tax_type TEXT NOT NULL DEFAULT '';
