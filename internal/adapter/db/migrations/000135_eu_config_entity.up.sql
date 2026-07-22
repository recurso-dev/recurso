-- Multi-Entity Books (Inc 3b, EU): the seller VAT/legal party on an EN 16931
-- e-invoice (Peppol/UBL) must be the ISSUING entity's registration. Mirrors the
-- India GST change (000134): tenant_eu_config was one row per tenant (tenant_id
-- PK); it becomes a tenant/primary default (entity_id NULL) plus one row per
-- non-primary entity. Resolution is STRICT — a non-primary entity with no EU
-- config of its own is not e-invoiced (the existing "no config" no-op), never
-- filed under the tenant default's VAT id. entity_id NULL = primary/tenant
-- default (canonical), so existing rows and single-entity tenants are unchanged.
ALTER TABLE tenant_eu_config DROP CONSTRAINT IF EXISTS tenant_eu_config_pkey;
ALTER TABLE tenant_eu_config ADD COLUMN entity_id UUID REFERENCES entities(id);
CREATE UNIQUE INDEX tenant_eu_config_default_key
  ON tenant_eu_config (tenant_id) WHERE entity_id IS NULL;
CREATE UNIQUE INDEX tenant_eu_config_entity_key
  ON tenant_eu_config (tenant_id, entity_id) WHERE entity_id IS NOT NULL;
