-- Quote money columns were INTEGER (int32, max ~2.14e9 minor units ≈ ₹21.4M /
-- $21.4M) while every other money column is BIGINT. A large enterprise quote
-- (e.g. a ₹30M annual contract) overflowed / hard-failed at the DB on insert
-- and carried a truncated value into the invoice it converts to. Widen to BIGINT.
ALTER TABLE quotes
  ALTER COLUMN subtotal TYPE BIGINT,
  ALTER COLUMN tax_amount TYPE BIGINT,
  ALTER COLUMN discount_amount TYPE BIGINT,
  ALTER COLUMN total TYPE BIGINT;

-- invoices.amount_remaining ignored credit_applied, so an invoice partially
-- settled by account credit reported its full (total - amount_paid) as
-- outstanding — overstating AR aging / DSO. Redefine to subtract applied credit.
-- A generated column's expression can't be altered in place, so drop + re-add;
-- the app selects invoice columns by name, so the changed column position is
-- harmless.
ALTER TABLE invoices DROP COLUMN IF EXISTS amount_remaining;
ALTER TABLE invoices ADD COLUMN amount_remaining BIGINT
  GENERATED ALWAYS AS (total - amount_paid - COALESCE(credit_applied, 0)) STORED;
