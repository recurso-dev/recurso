ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_type_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_type_check
    CHECK (type IN ('top_up', 'drain', 'expiry')) NOT VALID;
ALTER TABLE wallets DROP COLUMN IF EXISTS closed_at;
