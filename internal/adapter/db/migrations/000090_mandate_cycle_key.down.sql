DROP INDEX IF EXISTS idx_invoices_mandate_cycle_key;
ALTER TABLE invoices DROP COLUMN IF EXISTS mandate_cycle_key;
