-- Per-tenant US tax identity (W-9), kept isolated from the India GST config and
-- the EU config — the seller party shown on a US (sales-tax) invoice: the legal
-- name, the EIN (Employer Identification Number, the W-9 tax id), and a postal
-- address. Presentation only; it does not affect tax computation.
CREATE TABLE IF NOT EXISTS tenant_us_tax_config (
    tenant_id  UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    legal_name TEXT NOT NULL DEFAULT '',
    ein        TEXT NOT NULL DEFAULT '',
    address    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
