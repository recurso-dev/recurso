-- Multi-Entity Books (Inc 3b): the seller GSTIN on a government e-invoice (IRN)
-- must be the ISSUING entity's registration — never borrowed from another
-- entity. The India GST seller config was one row per tenant (tenant_id PK);
-- it becomes one tenant/primary default (entity_id NULL) plus one row per
-- non-primary entity that registers its own GSTIN. entity_id NULL = the
-- tenant/primary default (existing rows), matching the canonical primary=NULL
-- representation used across the epic. Resolution is STRICT: a non-primary
-- entity with no config of its own is skipped for e-invoicing (never mis-filed
-- under the tenant default's GSTIN).
ALTER TABLE tenant_gst_configs DROP CONSTRAINT IF EXISTS tenant_gst_configs_pkey;
ALTER TABLE tenant_gst_configs ADD COLUMN entity_id UUID REFERENCES entities(id);
CREATE UNIQUE INDEX tenant_gst_configs_default_key
  ON tenant_gst_configs (tenant_id) WHERE entity_id IS NULL;
CREATE UNIQUE INDEX tenant_gst_configs_entity_key
  ON tenant_gst_configs (tenant_id, entity_id) WHERE entity_id IS NOT NULL;
