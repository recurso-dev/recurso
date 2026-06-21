DROP INDEX IF EXISTS idx_invoices_dunning_managed;
ALTER TABLE invoices DROP COLUMN IF EXISTS dunning_managed_by;
ALTER TABLE invoices DROP COLUMN IF EXISTS last_payment_error;
ALTER TABLE invoices DROP COLUMN IF EXISTS dunning_context_key;
ALTER TABLE invoices DROP COLUMN IF EXISTS dunning_action_id;
