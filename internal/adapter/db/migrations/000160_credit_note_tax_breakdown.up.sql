-- B2 (ENG-196): store the tax breakdown on credit notes so the downloadable
-- credit-note document (#279) can render a statutory-grade CDN. For Indian
-- tenants a credit note must show the CGST/SGST/IGST breakup for the buyer to
-- reverse input tax; a gross-only document is not usable for compliance.
-- Populated at creation (downgrade credits carry the reversed proration tax;
-- refund credits derive proportionally from their invoice). subtotal > 0 marks
-- a breakdown as present; legacy rows keep 0/'' and render gross-only.
ALTER TABLE credit_notes
    ADD COLUMN IF NOT EXISTS subtotal    BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_amount  BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS igst_amount BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cgst_amount BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sgst_amount BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_type    VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS hsn_code    VARCHAR(16) NOT NULL DEFAULT '';
