-- BYO GoCardless (#237): the provider CHECK from 000107 predates the third
-- provider, so connecting GoCardless failed the insert with a 500.
ALTER TABLE gateway_connections DROP CONSTRAINT IF EXISTS gateway_connections_provider_check;
ALTER TABLE gateway_connections ADD CONSTRAINT gateway_connections_provider_check
    CHECK (provider IN ('stripe', 'razorpay', 'gocardless'));
