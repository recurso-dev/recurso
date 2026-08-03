-- Tenant-controlled invoice branding: the presentation layer of the invoice
-- PDF (display name, logo, signature, bank details, terms). Statutory seller
-- identity (GST / W-9 legal name) continues to live in its own settings and
-- takes precedence on tax invoices.
CREATE TABLE IF NOT EXISTS tenant_invoice_branding (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    company_name TEXT NOT NULL DEFAULT '',
    logo_data_url TEXT NOT NULL DEFAULT '',
    signature_data_url TEXT NOT NULL DEFAULT '',
    signatory_name TEXT NOT NULL DEFAULT '',
    bank_details TEXT NOT NULL DEFAULT '',
    terms TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
