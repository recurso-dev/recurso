DROP INDEX IF EXISTS tenant_eu_config_entity_key;
DROP INDEX IF EXISTS tenant_eu_config_default_key;
ALTER TABLE tenant_eu_config DROP COLUMN IF EXISTS entity_id;
ALTER TABLE tenant_eu_config ADD PRIMARY KEY (tenant_id);
