-- Remove e-invoice retry index
DROP INDEX IF EXISTS idx_invoices_einvoice_retry;

-- Remove new invoice columns
ALTER TABLE invoices DROP COLUMN IF EXISTS ack_date;
ALTER TABLE invoices DROP COLUMN IF EXISTS e_invoice_retry_count;
ALTER TABLE invoices DROP COLUMN IF EXISTS e_invoice_next_retry_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS e_invoice_error_message;

-- Drop tables
DROP TABLE IF EXISTS tenant_gst_configs;
DROP TABLE IF EXISTS tenant_irp_configs;

-- Note: Cannot remove enum values in PostgreSQL without recreating the type.
-- CANCELLED and NA values will remain in the enum.
