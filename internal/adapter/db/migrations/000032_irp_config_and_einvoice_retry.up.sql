-- Add CANCELLED and NA to e_invoice_status enum (if not already present)
DO $$
BEGIN
    -- Check if type exists before altering
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'e_invoice_status') THEN
        BEGIN
            ALTER TYPE e_invoice_status ADD VALUE IF NOT EXISTS 'CANCELLED';
        EXCEPTION WHEN duplicate_object THEN NULL;
        END;
        BEGIN
            ALTER TYPE e_invoice_status ADD VALUE IF NOT EXISTS 'NA';
        EXCEPTION WHEN duplicate_object THEN NULL;
        END;
    END IF;
END$$;

-- Create tenant_irp_configs table for per-tenant IRP credentials
CREATE TABLE IF NOT EXISTS tenant_irp_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    environment VARCHAR(20) NOT NULL DEFAULT 'sandbox' CHECK (environment IN ('sandbox', 'production')),
    client_id VARCHAR(255) NOT NULL DEFAULT '',
    client_secret VARCHAR(500) NOT NULL DEFAULT '',
    username VARCHAR(255) NOT NULL DEFAULT '',
    password VARCHAR(500) NOT NULL DEFAULT '',
    gstin VARCHAR(15) NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, environment)
);

-- Create tenant_gst_configs table to persist GST config (replacing in-memory)
CREATE TABLE IF NOT EXISTS tenant_gst_configs (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    gstin VARCHAR(15) NOT NULL DEFAULT '',
    state_code VARCHAR(2) NOT NULL DEFAULT '',
    state_name VARCHAR(100) NOT NULL DEFAULT '',
    sac_code VARCHAR(10) NOT NULL DEFAULT '998314',
    gst_rate NUMERIC(5,2) NOT NULL DEFAULT 18.00,
    pan VARCHAR(10) NOT NULL DEFAULT '',
    legal_name VARCHAR(255) NOT NULL DEFAULT '',
    trade_name VARCHAR(255) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    has_lut BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add new columns to invoices for e-invoice retry tracking
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS ack_date VARCHAR(50) DEFAULT '';
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS e_invoice_retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS e_invoice_next_retry_at TIMESTAMP;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS e_invoice_error_message TEXT DEFAULT '';

-- Index for e-invoice retry worker: find FAILED invoices due for retry
CREATE INDEX IF NOT EXISTS idx_invoices_einvoice_retry
    ON invoices (e_invoice_next_retry_at)
    WHERE e_invoice_status = 'FAILED' AND e_invoice_next_retry_at IS NOT NULL;
