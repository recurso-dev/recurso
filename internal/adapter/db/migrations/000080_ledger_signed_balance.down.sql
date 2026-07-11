-- Best-effort revert to the old accumulate-only semantics: balance is the gross
-- amount that flowed through the account (debits + credits). Reconstructed from
-- the rebuilt posted totals; the exact pre-migration bytes cannot be restored.
UPDATE ledger_accounts SET
    balance = debits_posted + credits_posted,
    updated_at = NOW();
