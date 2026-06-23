CREATE TABLE cancel_flows (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    cooldown_days INT DEFAULT 30,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE cancel_flow_steps (
    id UUID PRIMARY KEY,
    flow_id UUID NOT NULL REFERENCES cancel_flows(id) ON DELETE CASCADE,
    step_order INT NOT NULL,
    step_type VARCHAR(20) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE cancel_flow_sessions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    subscription_id UUID NOT NULL,
    flow_id UUID NOT NULL REFERENCES cancel_flows(id),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    current_step_index INT DEFAULT 0,
    cancellation_reason VARCHAR(50),
    feedback TEXT,
    offer_presented JSONB,
    offer_accepted BOOLEAN DEFAULT FALSE,
    saved_by_offer BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_cancel_sessions_customer ON cancel_flow_sessions(customer_id, started_at DESC);
CREATE INDEX idx_cancel_sessions_tenant ON cancel_flow_sessions(tenant_id, status);
