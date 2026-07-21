ALTER TABLE customers
    DROP COLUMN IF EXISTS tax_exemption_code,
    DROP COLUMN IF EXISTS tax_exemption_number,
    DROP COLUMN IF EXISTS tax_exempt;
