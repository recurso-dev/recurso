-- B1 autopay (BYO): a saved payment method's pm_* token is only chargeable on
-- the gateway account that created it. Record which gateway connection saved the
-- card so off-session charges (renewal, ...) route to the SAME gateway.
--
-- NULL = saved on the platform gateway (all pre-B1 cards, and tenants without a
-- BYO connection) — those keep charging on the platform, unchanged. A non-null
-- value points at the tenant's BYO connection the card was saved on.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS pm_gateway_connection_id UUID REFERENCES gateway_connections(id) ON DELETE SET NULL;
