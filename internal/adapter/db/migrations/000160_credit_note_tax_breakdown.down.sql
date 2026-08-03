ALTER TABLE credit_notes
    DROP COLUMN IF EXISTS subtotal,
    DROP COLUMN IF EXISTS tax_amount,
    DROP COLUMN IF EXISTS igst_amount,
    DROP COLUMN IF EXISTS cgst_amount,
    DROP COLUMN IF EXISTS sgst_amount,
    DROP COLUMN IF EXISTS tax_type,
    DROP COLUMN IF EXISTS hsn_code;
