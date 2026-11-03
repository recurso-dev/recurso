DROP INDEX IF EXISTS idx_invoices_renewal_cycle_key;
ALTER TABLE invoices DROP COLUMN IF EXISTS renewal_cycle_key;
