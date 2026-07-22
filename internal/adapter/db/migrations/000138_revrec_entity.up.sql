-- Multi-Entity Books: revenue recognition must post to the invoice's legal
-- entity, not always the primary. Before this, RecordRecognition hardcoded the
-- primary ledger, so a non-primary entity's Deferred Revenue grew at invoice
-- time (RecordInvoice is entity-aware) but never drained — its recognition legs
-- landed on the primary entity's P&L instead.
--
-- entity_id mirrors the canonical convention (NULL ⇒ primary entity); backfill
-- from the schedule's invoice, whose entity_id is already correct.
ALTER TABLE revenue_schedules ADD COLUMN entity_id UUID REFERENCES entities(id);

UPDATE revenue_schedules rs
   SET entity_id = i.entity_id
  FROM invoices i
 WHERE i.id = rs.invoice_id
   AND rs.entity_id IS NULL;

CREATE INDEX idx_revenue_schedules_entity ON revenue_schedules (entity_id);
