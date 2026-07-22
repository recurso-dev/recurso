-- Wallet closure / cash-out (completes the prepaid-wallet lifecycle): a closed
-- wallet has had its remaining balance settled (paid residue refunded to the
-- customer, promotional residue forfeited) and accepts no further top-ups or
-- drains. NULL = open (the default), so existing wallets are unchanged.
ALTER TABLE wallets ADD COLUMN closed_at TIMESTAMPTZ;

-- Closure adds two movement types: 'refund' (paid residue returned) and
-- 'forfeit' (promotional residue written off). Widen the wallet_transactions
-- type CHECK to allow them; existing rows (top_up/drain/expiry) satisfy the
-- superset, so this validates cleanly.
ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_type_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_type_check
    CHECK (type IN ('top_up', 'drain', 'expiry', 'refund', 'forfeit'));
