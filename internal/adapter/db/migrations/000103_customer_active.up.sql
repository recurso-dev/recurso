-- Customers gain a soft-archive flag. Archiving is blocked while the
-- customer has active subscriptions; archived customers keep their full
-- billing history and can be restored.
ALTER TABLE customers ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
