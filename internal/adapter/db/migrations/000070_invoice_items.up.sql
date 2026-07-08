-- Per-product HSN codes & itemized invoice tax (Phase 1: itemize, no rate change).
-- Each invoice line records its own HSN/SAC code and per-line GST breakdown.
-- Phase 1 lines all use the tenant SAC, so invoice totals are unchanged.
CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL DEFAULT '',
    hsn_code VARCHAR(20) NOT NULL DEFAULT '',
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_amount BIGINT NOT NULL DEFAULT 0,
    amount BIGINT NOT NULL DEFAULT 0,
    tax_rate NUMERIC(6,3) NOT NULL DEFAULT 0,
    cgst_amount BIGINT NOT NULL DEFAULT 0,
    sgst_amount BIGINT NOT NULL DEFAULT 0,
    igst_amount BIGINT NOT NULL DEFAULT 0,
    taxable_amount BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items(invoice_id);
