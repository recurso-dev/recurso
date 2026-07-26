-- Collections Intelligence Inc 3: let an operator pause automated dunning on a
-- single invoice (retry worker + email scheduler both skip paused rows) without
-- losing its place in the queue. Default FALSE keeps every existing invoice
-- exactly as it was.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS dunning_paused BOOLEAN NOT NULL DEFAULT FALSE;
