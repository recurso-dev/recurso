ALTER TABLE gateway_connections DROP CONSTRAINT IF EXISTS gateway_connections_provider_check;
ALTER TABLE gateway_connections ADD CONSTRAINT gateway_connections_provider_check
    CHECK (provider IN ('stripe', 'razorpay'));
