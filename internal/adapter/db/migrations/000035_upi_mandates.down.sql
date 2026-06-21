ALTER TABLE subscriptions DROP COLUMN IF EXISTS mandate_id;
DROP INDEX IF EXISTS idx_mandates_customer;
DROP INDEX IF EXISTS idx_mandates_tenant;
DROP INDEX IF EXISTS idx_mandates_next_debit;
DROP TABLE IF EXISTS mandates;
