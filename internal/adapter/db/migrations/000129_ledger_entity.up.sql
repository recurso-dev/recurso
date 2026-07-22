-- Multi-Entity Books (Inc 2a): make the ledger entity-aware. Each entity keeps
-- its own chart of accounts (and its own AR sub-ledger) on its TigerBeetle
-- ledger id. Existing accounts + invoices are backfilled to each tenant's
-- primary entity, whose tb_ledger_id is 1 — so the primary entity's books are
-- byte-identical to today and the ledger invariant harness stays green.
ALTER TABLE ledger_accounts ADD COLUMN entity_id UUID REFERENCES entities(id);
ALTER TABLE invoices        ADD COLUMN entity_id UUID REFERENCES entities(id);

UPDATE ledger_accounts la SET entity_id = e.id
  FROM entities e WHERE e.tenant_id = la.tenant_id AND e.is_primary;
UPDATE invoices i SET entity_id = e.id
  FROM entities e WHERE e.tenant_id = i.tenant_id AND e.is_primary;

-- GL accounts (Cash, Revenue, …) are one per (tenant, entity, code); AR (code
-- 1100) repeats per customer, so this index is non-unique and just speeds the
-- per-entity account lookup.
CREATE INDEX idx_ledger_accounts_entity_code ON ledger_accounts (tenant_id, entity_id, code);
CREATE INDEX idx_invoices_entity ON invoices (entity_id);
