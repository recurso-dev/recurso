DROP INDEX IF EXISTS idx_invoices_entity;
DROP INDEX IF EXISTS idx_ledger_accounts_entity_code;
ALTER TABLE invoices        DROP COLUMN IF EXISTS entity_id;
ALTER TABLE ledger_accounts DROP COLUMN IF EXISTS entity_id;
