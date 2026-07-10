-- API keys carry a livemode flag so test keys (rsk_test_) can be gated away
-- from live-money servers. Existing keys are grandfathered by prefix: any
-- *_test key is test-mode, everything else (sk_live_, rsk_live_) is live.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS livemode BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE api_keys SET livemode = FALSE
    WHERE key_prefix ILIKE 'sk_test%' OR key_prefix ILIKE 'rsk_test%';
