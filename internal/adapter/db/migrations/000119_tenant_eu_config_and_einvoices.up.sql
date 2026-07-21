-- EU e-invoicing (Track C, increment 1b). Regional compliance data is kept
-- strictly isolated from the India GST config and the generic tenant row: a
-- dedicated per-tenant EU seller identity + an opt-in flag, plus a table holding
-- each invoice's generated EN 16931 document and its delivery status.

CREATE TABLE IF NOT EXISTS tenant_eu_config (
    tenant_id    UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    -- Opt-in: EU e-invoice generation fires only when a tenant enables it, since
    -- the EU mandate landscape is fragmented (B2G vs B2B, per-country rollout).
    enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    -- Seller party (EN 16931): registered name (BT-27), VAT id incl. country
    -- prefix (BT-31), and a structured postal address (BG-5).
    legal_name   TEXT NOT NULL DEFAULT '',
    vat_number   TEXT NOT NULL DEFAULT '',
    country_code TEXT NOT NULL DEFAULT '',
    street       TEXT NOT NULL DEFAULT '',
    city         TEXT NOT NULL DEFAULT '',
    postal_zone  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS eu_einvoices (
    id            UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- One structured e-invoice per invoice; the UNIQUE makes generation
    -- idempotent (upsert on re-run rather than a duplicate row).
    invoice_id    UUID NOT NULL UNIQUE REFERENCES invoices(id) ON DELETE CASCADE,
    syntax        TEXT NOT NULL DEFAULT 'ubl21',
    status        TEXT NOT NULL CHECK (status IN ('generated','sent','failed')),
    document      TEXT NOT NULL DEFAULT '',   -- the serialized EN 16931 document (UBL XML)
    message_id    TEXT NOT NULL DEFAULT '',   -- transport (Access Point) message id
    error_message TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eu_einvoices_tenant ON eu_einvoices (tenant_id);
