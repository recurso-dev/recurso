-- Gift cancellation (account-credit policy). Links each gift to its buyer
-- purchase invoice so cancel can act on what actually happened: a PAID
-- purchase is credited back as spendable account credit; a still-open one is
-- simply voided. Nullable — gifts purchased before this column have no link
-- (cancel then flips status only and the operator issues any credit manually).
ALTER TABLE gifts ADD COLUMN IF NOT EXISTS invoice_id UUID REFERENCES invoices(id);
