ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_tenant_customer_entity_currency_key;
ALTER TABLE wallets ADD CONSTRAINT wallets_tenant_id_customer_id_currency_key
  UNIQUE (tenant_id, customer_id, currency);
ALTER TABLE wallets DROP COLUMN IF EXISTS entity_id;
