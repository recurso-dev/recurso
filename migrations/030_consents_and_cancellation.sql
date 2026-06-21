-- Migration: Create consents table for RBI compliance
-- This tracks customer consent for recurring billing

CREATE TABLE IF NOT EXISTS consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    subscription_id UUID REFERENCES subscriptions(id),
    consent_type VARCHAR(50) NOT NULL,
    
    -- Consent status
    granted BOOLEAN NOT NULL DEFAULT true,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit trail
    ip_address VARCHAR(45),
    user_agent TEXT,
    consent_text TEXT NOT NULL,
    version VARCHAR(20) NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_consents_tenant_customer ON consents(tenant_id, customer_id);
CREATE INDEX idx_consents_subscription ON consents(subscription_id) WHERE subscription_id IS NOT NULL;
CREATE INDEX idx_consents_type ON consents(consent_type, granted);

-- Add cancellation fields to subscriptions table
ALTER TABLE subscriptions 
ADD COLUMN IF NOT EXISTS cancellation_reason VARCHAR(100),
ADD COLUMN IF NOT EXISTS cancellation_feedback TEXT;
