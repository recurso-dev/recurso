-- Multi-Entity Books (Inc 2d): wallets are entity-scoped — a prepaid balance
-- belongs to a (customer, entity, currency) and is spendable only on that
-- entity's invoices (decision: entity-scoped balances). Existing wallets
-- backfill to each tenant's primary entity, so single-entity tenants are
-- unchanged. The uniqueness moves from (tenant, customer, currency) to
-- (tenant, customer, entity, currency) so a customer can hold one wallet per
-- entity per currency.
ALTER TABLE wallets ADD COLUMN entity_id UUID REFERENCES entities(id);
UPDATE wallets w SET entity_id = e.id
  FROM entities e WHERE e.tenant_id = w.tenant_id AND e.is_primary;
ALTER TABLE wallets ALTER COLUMN entity_id SET NOT NULL;

ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_tenant_id_customer_id_currency_key;
ALTER TABLE wallets ADD CONSTRAINT wallets_tenant_customer_entity_currency_key
  UNIQUE (tenant_id, customer_id, entity_id, currency);
