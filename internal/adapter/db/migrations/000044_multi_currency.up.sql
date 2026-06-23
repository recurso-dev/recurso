-- Invoice FX fields
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS exchange_rate DOUBLE PRECISION DEFAULT 1.0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS base_currency_total BIGINT DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS base_currency CHAR(3) DEFAULT 'USD';

-- Tenant base currency
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS base_currency CHAR(3) DEFAULT 'USD';
