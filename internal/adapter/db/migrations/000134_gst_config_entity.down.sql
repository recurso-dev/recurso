DROP INDEX IF EXISTS tenant_gst_configs_entity_key;
DROP INDEX IF EXISTS tenant_gst_configs_default_key;
ALTER TABLE tenant_gst_configs DROP COLUMN IF EXISTS entity_id;
ALTER TABLE tenant_gst_configs ADD PRIMARY KEY (tenant_id);
