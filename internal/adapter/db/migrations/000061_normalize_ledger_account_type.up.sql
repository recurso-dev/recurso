-- Older code versions wrote human-readable words ("asset") into the
-- ledger_accounts.type varchar; the current model uses numeric codes.
-- Normalize any legacy string words to the canonical numeric codes.
UPDATE ledger_accounts SET type = '1' WHERE lower(type) = 'asset';
UPDATE ledger_accounts SET type = '2' WHERE lower(type) = 'liability';
UPDATE ledger_accounts SET type = '3' WHERE lower(type) = 'equity';
UPDATE ledger_accounts SET type = '4' WHERE lower(type) = 'revenue';
UPDATE ledger_accounts SET type = '5' WHERE lower(type) = 'expense';
