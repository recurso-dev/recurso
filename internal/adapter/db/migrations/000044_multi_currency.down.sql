ALTER TABLE invoices DROP COLUMN IF EXISTS exchange_rate;
ALTER TABLE invoices DROP COLUMN IF EXISTS base_currency_total;
ALTER TABLE invoices DROP COLUMN IF EXISTS base_currency;
ALTER TABLE tenants DROP COLUMN IF EXISTS base_currency;
