-- Per-tenant MCP server opt-in. tier3_enabled gates the money-path/destructive
-- MCP tools (convert quote→invoice, cancel subscription, credit note, wallet
-- top-up, …), which stay OFF unless a tenant explicitly enables them.
CREATE TABLE mcp_settings (
    tenant_id     UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    tier3_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
