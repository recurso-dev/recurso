DROP INDEX IF EXISTS tenant_tax_nexus_entity_key;
DROP INDEX IF EXISTS tenant_tax_nexus_default_key;
ALTER TABLE tenant_tax_nexus DROP COLUMN IF EXISTS entity_id;
ALTER TABLE tenant_tax_nexus ADD CONSTRAINT tenant_tax_nexus_tenant_id_state_code_key UNIQUE (tenant_id, state_code);

DROP INDEX IF EXISTS tenant_irp_configs_entity_key;
DROP INDEX IF EXISTS tenant_irp_configs_default_key;
ALTER TABLE tenant_irp_configs DROP COLUMN IF EXISTS entity_id;
ALTER TABLE tenant_irp_configs ADD CONSTRAINT tenant_irp_configs_tenant_id_environment_key UNIQUE (tenant_id, environment);
