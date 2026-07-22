-- Multi-Entity Books (Inc 1): legal entities under a tenant. Each entity gets
-- its own TigerBeetle ledger (tb_ledger_id) and its own gapless invoice series
-- (entity_invoice_sequences). Every tenant has exactly one primary entity on
-- tb_ledger_id = 1 — existing tenants are backfilled here; new tenants get one
-- via a trigger. So single-entity tenants (all of them today) are unchanged.
-- Nothing reads these tables yet; this increment is pure structure.
CREATE TABLE entities (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    legal_name     TEXT NOT NULL DEFAULT '',
    is_primary     BOOLEAN NOT NULL DEFAULT FALSE,
    tb_ledger_id   INTEGER NOT NULL,
    invoice_prefix TEXT NOT NULL DEFAULT 'INV',
    country_code   TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Exactly one primary entity per tenant; ledger ids unique within a tenant.
CREATE UNIQUE INDEX idx_entities_one_primary ON entities (tenant_id) WHERE is_primary;
CREATE UNIQUE INDEX idx_entities_tenant_ledger ON entities (tenant_id, tb_ledger_id);
CREATE INDEX idx_entities_tenant ON entities (tenant_id);

-- Gapless per-entity invoice counter (drawn at finalization in a later increment).
CREATE TABLE entity_invoice_sequences (
    entity_id   UUID PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    next_number BIGINT NOT NULL DEFAULT 1
);

-- Backfill existing tenants: one primary entity each, on the existing LedgerID = 1.
INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
SELECT id, name, name, TRUE, 1, 'INV' FROM tenants;
INSERT INTO entity_invoice_sequences (entity_id, next_number)
SELECT id, 1 FROM entities;

-- New tenants get their primary entity + sequence automatically, from any code
-- path — the guarantee "every tenant has exactly one primary entity" lives in
-- the database, not in each registration flow.
CREATE OR REPLACE FUNCTION create_primary_entity() RETURNS TRIGGER AS $$
DECLARE new_entity_id UUID;
BEGIN
    INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
    VALUES (NEW.id, NEW.name, NEW.name, TRUE, 1, 'INV')
    RETURNING id INTO new_entity_id;
    INSERT INTO entity_invoice_sequences (entity_id, next_number) VALUES (new_entity_id, 1);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenant_primary_entity
    AFTER INSERT ON tenants
    FOR EACH ROW EXECUTE FUNCTION create_primary_entity();
