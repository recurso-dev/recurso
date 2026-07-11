-- ENG-148: ledger_accounts.balance was accumulate-only. CreateTransaction ran
-- `balance = balance + amount` on BOTH the debit and the credit account, so
-- `balance` was the gross amount that had flowed through the account (either
-- side), never a signed double-entry balance, and the debits_posted /
-- credits_posted columns went unused. A trial balance pulled from `balance`
-- did not balance.
--
-- Rebuild the posted totals from the transaction log (the source of truth),
-- then derive a SIGNED balance by account normal side:
--   assets/expenses (type 1/5)            = debits_posted - credits_posted
--   liabilities/equity/revenue (2/3/4)    = credits_posted - debits_posted
-- The lower(type) IN (...) predicate also tolerates any legacy word-form rows.

UPDATE ledger_accounts la SET
    debits_posted  = COALESCE((SELECT SUM(amount) FROM ledger_transactions WHERE debit_account_id  = la.id), 0),
    credits_posted = COALESCE((SELECT SUM(amount) FROM ledger_transactions WHERE credit_account_id = la.id), 0);

UPDATE ledger_accounts SET
    balance = CASE WHEN lower(type) IN ('1', '5', 'asset', 'expense')
                   THEN debits_posted - credits_posted
                   ELSE credits_posted - debits_posted END,
    updated_at = NOW();
