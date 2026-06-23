CREATE TABLE dunning_campaigns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    trigger_event VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE dunning_campaign_steps (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES dunning_campaigns(id) ON DELETE CASCADE,
    step_order INT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    delay_hours INT NOT NULL DEFAULT 0,
    template_name VARCHAR(100),
    subject VARCHAR(200),
    body TEXT,
    is_payment_wall BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE dunning_campaign_executions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    invoice_id UUID NOT NULL,
    campaign_id UUID NOT NULL REFERENCES dunning_campaigns(id),
    current_step_index INT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ DEFAULT NOW(),
    next_step_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE(invoice_id, campaign_id)
);

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_wall_active BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_dunning_executions_due ON dunning_campaign_executions(next_step_at) WHERE status = 'active';
CREATE INDEX idx_dunning_executions_invoice ON dunning_campaign_executions(invoice_id);
