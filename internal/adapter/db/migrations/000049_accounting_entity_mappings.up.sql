CREATE TABLE IF NOT EXISTS accounting_entity_mappings (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    connection_id UUID NOT NULL REFERENCES accounting_connections(id),
    entity_type VARCHAR(30) NOT NULL,
    entity_id UUID NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(connection_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_accounting_entity_mappings_tenant ON accounting_entity_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_accounting_entity_mappings_entity ON accounting_entity_mappings(entity_type, entity_id);
