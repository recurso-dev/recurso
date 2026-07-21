-- US sales-tax exemption on the customer (Track D · D2). An exempt buyer (resale,
-- government, non-profit, ...) is passed through to the tax provider with its
-- exemption number and entity-use code so the provider returns zero tax AND
-- records an exempt sale for the tenant's liability reporting — rather than the
-- engine short-circuiting and the sale going unrecorded.

ALTER TABLE customers
    -- Exemption status. When true, the number/code below are passed to the
    -- provider on every US sales-tax lookup for this customer.
    ADD COLUMN IF NOT EXISTS tax_exempt           BOOLEAN NOT NULL DEFAULT FALSE,
    -- Exemption / resale certificate number on file.
    ADD COLUMN IF NOT EXISTS tax_exemption_number TEXT    NOT NULL DEFAULT '',
    -- Provider entity-use / customer-usage code (e.g. Avalara "A" = federal govt,
    -- "E" = charitable). Doubles as the exemption reason.
    ADD COLUMN IF NOT EXISTS tax_exemption_code   TEXT    NOT NULL DEFAULT '';
