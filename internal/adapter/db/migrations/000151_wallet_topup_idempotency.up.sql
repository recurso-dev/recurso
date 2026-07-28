-- Auto-recharge double-credit guard: ListDueForRecharge has no atomic claim
-- (unlike every sibling worker), and TopUp was not idempotent — under a
-- Redis-less multi-instance deploy two concurrent sweeps charged the card
-- once (Stripe idempotency key) but credited the wallet twice. A unique key
-- on the top-up makes the second credit conflict and no-op.
ALTER TABLE wallet_transactions ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_tx_idempotency_key
    ON wallet_transactions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
