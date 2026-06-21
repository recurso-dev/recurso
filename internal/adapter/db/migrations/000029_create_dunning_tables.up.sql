-- Persistence for learned weights (Bandit State)
CREATE TABLE IF NOT EXISTS dunning_weights (
    context_key VARCHAR(100) NOT NULL,
    action_id VARCHAR(20) NOT NULL,
    average_reward DOUBLE PRECISION DEFAULT 0.0,
    sample_count BIGINT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (context_key, action_id)
);

-- Audit trail for retry outcomes
CREATE TABLE IF NOT EXISTS dunning_history (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    context_key VARCHAR(100) NOT NULL,
    action_id VARCHAR(20) NOT NULL,
    retry_interval INTEGER NOT NULL, -- seconds
    outcome VARCHAR(20),             -- 'success', 'failure'
    reward DOUBLE PRECISION,        -- 1.0, 0.0
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dunning_history_invoice ON dunning_history(invoice_id);
CREATE INDEX idx_dunning_history_context ON dunning_history(context_key);
