-- Multi-Entity Books (Inc 2e): a credit note is issued by a legal entity — its
-- Customer-Credit liability posts on that entity's ledger and (once the
-- application side lands) it is spendable only on that entity's invoices.
-- Nullable mirrors invoices/subscriptions, where the PRIMARY entity is
-- represented as NULL (not a concrete id) so its books stay byte-identical.
-- Backfill only non-primary credits (inherit the referenced invoice's entity);
-- primary/standalone credits stay NULL, matching how issuance stores them
-- (cn.EntityID = inv.EntityID / sub.EntityID, which is NULL for primary).
ALTER TABLE credit_notes ADD COLUMN entity_id UUID REFERENCES entities(id);

UPDATE credit_notes cn SET entity_id = i.entity_id
  FROM invoices i
  WHERE cn.invoice_id = i.id AND i.entity_id IS NOT NULL;
