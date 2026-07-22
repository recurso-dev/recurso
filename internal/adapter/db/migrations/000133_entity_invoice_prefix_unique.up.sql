-- Multi-Entity Books (Inc 3a): each entity issues its own gapless invoice
-- series numbered {invoice_prefix}-{seq:06d} drawn from entity_invoice_sequences.
-- The per-entity sequence already guarantees uniqueness WITHIN an issuer; this
-- index keeps invoice prefixes distinct per tenant so two entities can't emit
-- visually identical numbers (e.g. two "INV-000001"). Existing tenants each have
-- exactly one primary entity ('INV'), so no collision exists to migrate.
CREATE UNIQUE INDEX entities_tenant_prefix_key
  ON entities (tenant_id, upper(invoice_prefix));
