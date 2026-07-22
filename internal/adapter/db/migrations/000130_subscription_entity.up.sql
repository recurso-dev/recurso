-- Multi-Entity Books (Inc 2b): subscriptions carry the issuing legal entity, so
-- their invoices inherit it and post to that entity's ledger. Nullable +
-- backfilled to each tenant's primary entity; NULL means "primary" everywhere
-- it's read, so single-entity tenants are unchanged.
ALTER TABLE subscriptions ADD COLUMN entity_id UUID REFERENCES entities(id);
UPDATE subscriptions s SET entity_id = e.id
  FROM entities e WHERE e.tenant_id = s.tenant_id AND e.is_primary;
CREATE INDEX idx_subscriptions_entity ON subscriptions (entity_id);
