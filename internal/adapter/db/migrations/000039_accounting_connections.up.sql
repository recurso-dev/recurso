CREATE TABLE IF NOT EXISTS accounting_connections (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider VARCHAR(20) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    realm_id VARCHAR(100),
    last_sync_at TIMESTAMPTZ,
    sync_status VARCHAR(20) DEFAULT 'idle',
    last_error TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, provider)
);

CREATE TABLE IF NOT EXISTS accounting_sync_log (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    connection_id UUID NOT NULL REFERENCES accounting_connections(id),
    entity_type VARCHAR(30) NOT NULL,
    entity_id UUID NOT NULL,
    external_id VARCHAR(100),
    action VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    synced_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounting_connections_tenant ON accounting_connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_accounting_sync_log_connection ON accounting_sync_log(connection_id);
