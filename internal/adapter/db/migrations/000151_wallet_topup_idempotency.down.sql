DROP INDEX IF EXISTS idx_wallet_tx_idempotency_key;
ALTER TABLE wallet_transactions DROP COLUMN IF EXISTS idempotency_key;
