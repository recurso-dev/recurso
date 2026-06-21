-- Revenue Recognition Tables
CREATE TABLE IF NOT EXISTS revenue_schedules (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    subscription_id UUID REFERENCES subscriptions(id),
    total_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recognition_events (
    id UUID PRIMARY KEY,
    revenue_schedule_id UUID NOT NULL REFERENCES revenue_schedules(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    amount BIGINT NOT NULL,
    recognition_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    ledger_tx_id UUID, -- Optional: link to TigerBeetle transfer if executed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_revrec_schedules_tenant ON revenue_schedules(tenant_id);
CREATE INDEX idx_revrec_events_date ON recognition_events(recognition_date) WHERE status = 'pending';
CREATE INDEX idx_revrec_events_schedule ON recognition_events(revenue_schedule_id);
