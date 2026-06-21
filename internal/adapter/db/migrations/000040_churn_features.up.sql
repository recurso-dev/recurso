CREATE TABLE IF NOT EXISTS churn_feature_snapshots (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    days_since_signup INT,
    total_invoices INT,
    failed_invoices_90d INT,
    payment_failure_rate DECIMAL(5,4),
    avg_days_to_pay DECIMAL(8,2),
    plan_downgrades INT,
    months_active INT,
    current_mrr BIGINT,
    usage_trend DECIMAL(5,4),
    risk_score INT NOT NULL,
    model_version VARCHAR(20) DEFAULT 'v1',
    computed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS churn_alerts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    previous_score INT,
    new_score INT,
    threshold INT,
    alert_type VARCHAR(20),
    acknowledged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_churn_snapshots_customer ON churn_feature_snapshots(customer_id);
CREATE INDEX IF NOT EXISTS idx_churn_snapshots_tenant ON churn_feature_snapshots(tenant_id);
CREATE INDEX IF NOT EXISTS idx_churn_alerts_tenant ON churn_alerts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_churn_alerts_unack ON churn_alerts(tenant_id) WHERE acknowledged = FALSE;
