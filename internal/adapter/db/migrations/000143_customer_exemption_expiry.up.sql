-- US sales-tax exemption expiry (Inc 2). A resale/exemption certificate is valid
-- only through its expiry date; past that the buyer must be charged tax again.
-- NULL means "no expiry on file" (the certificate does not expire / not tracked),
-- which preserves today's always-honored behavior for existing exempt customers.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS tax_exemption_expires_at DATE;
