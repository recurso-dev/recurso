-- Caller-supplied idempotency key for invoices created through the manual
-- (explicit line-item) invoice API, e.g. "<billing_entity>|<company>|2026-08".
-- NULL for every subscription/quote/gift invoice, which the partial unique
-- index leaves unconstrained. A repeated key is a no-op replay, never a
-- second invoice.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS external_ref TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoices_tenant_external_ref
    ON invoices (tenant_id, external_ref)
    WHERE external_ref IS NOT NULL;
