-- Multi-Entity Books (Inc 3b, IRP + US nexus): the last two tax backends adopt
-- the per-entity schema pattern (entity_id NULL = tenant/primary default, plus
-- one row set per non-primary entity), matching tenant_gst_configs (000134) and
-- tenant_eu_config (000135).

-- IRP submission credentials: each issuing entity submits under its own NIC/IRP
-- account. Was UNIQUE(tenant_id, environment); now one default per environment
-- plus one per (tenant, environment, entity). nic.go resolves the invoice's
-- entity, so a non-primary entity submits under its own credentials.
ALTER TABLE tenant_irp_configs DROP CONSTRAINT IF EXISTS tenant_irp_configs_tenant_id_environment_key;
ALTER TABLE tenant_irp_configs ADD COLUMN entity_id UUID REFERENCES entities(id);
CREATE UNIQUE INDEX tenant_irp_configs_default_key
  ON tenant_irp_configs (tenant_id, environment) WHERE entity_id IS NULL;
CREATE UNIQUE INDEX tenant_irp_configs_entity_key
  ON tenant_irp_configs (tenant_id, environment, entity_id) WHERE entity_id IS NOT NULL;

-- US sales-tax nexus: nexus states become per issuing entity. Tax CALCULATION
-- still asks the provider (Avalara/TaxJar) for nexus, so there is no resolution
-- change here — this is the schema + repo scoping so the tenant/primary nexus
-- set (entity_id NULL) is managed independently of any future per-entity sets.
ALTER TABLE tenant_tax_nexus DROP CONSTRAINT IF EXISTS tenant_tax_nexus_tenant_id_state_code_key;
ALTER TABLE tenant_tax_nexus ADD COLUMN entity_id UUID REFERENCES entities(id);
CREATE UNIQUE INDEX tenant_tax_nexus_default_key
  ON tenant_tax_nexus (tenant_id, state_code) WHERE entity_id IS NULL;
CREATE UNIQUE INDEX tenant_tax_nexus_entity_key
  ON tenant_tax_nexus (tenant_id, entity_id, state_code) WHERE entity_id IS NOT NULL;
