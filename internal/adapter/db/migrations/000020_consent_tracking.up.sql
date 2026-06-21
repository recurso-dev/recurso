-- Consent tracking for recurring billing compliance
CREATE TABLE IF NOT EXISTS consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    
    consent_type VARCHAR(50) NOT NULL,  -- 'recurring_billing', 'email_marketing', etc.
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit trail
    ip_address VARCHAR(45),  -- IPv6 max length
    user_agent TEXT,
    consent_text TEXT NOT NULL,  -- Exact text shown to user
    version VARCHAR(20) NOT NULL,  -- Version of consent text
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_consents_customer ON consents(customer_id);
CREATE INDEX idx_consents_subscription ON consents(subscription_id);
CREATE INDEX idx_consents_type ON consents(consent_type);
CREATE INDEX idx_consents_granted ON consents(granted) WHERE granted = true;

-- Add GST fields to customers table if not exists
ALTER TABLE customers ADD COLUMN IF NOT EXISTS gstin VARCHAR(15);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS state_code VARCHAR(2);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS place_of_supply VARCHAR(50);

-- Add seller GST config to tenants
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gstin VARCHAR(15);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS state_code VARCHAR(2);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS sac_code VARCHAR(10) DEFAULT '998314';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gst_rate DECIMAL(5,2) DEFAULT 18.00;

-- Pre-charge notification tracking
CREATE TABLE IF NOT EXISTS precharge_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    
    scheduled_charge_date DATE NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    
    notification_sent_at TIMESTAMP WITH TIME ZONE,
    notification_type VARCHAR(20) DEFAULT 'email',  -- 'email', 'sms', 'push'
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_precharge_subscription ON precharge_notifications(subscription_id);
CREATE INDEX idx_precharge_charge_date ON precharge_notifications(scheduled_charge_date);
CREATE INDEX idx_precharge_not_sent ON precharge_notifications(id) WHERE notification_sent_at IS NULL;

COMMENT ON TABLE consents IS 'Tracks user consent for recurring billing (RBI/FTC compliance)';
COMMENT ON TABLE precharge_notifications IS 'Tracks 24-hour pre-charge notifications (RBI mandate compliance)';
