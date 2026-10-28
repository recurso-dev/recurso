DROP INDEX IF EXISTS uq_invoices_tenant_external_ref;
ALTER TABLE invoices DROP COLUMN IF EXISTS external_ref;
