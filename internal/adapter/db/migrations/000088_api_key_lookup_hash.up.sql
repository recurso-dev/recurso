-- key_lookup is the SHA-256 (hex) of the full API key, enabling an O(1)
-- exact-match lookup at authentication time.
--
-- The existing key_prefix is only the first 8 chars of the key, but every key is
-- "rsk_live_<uuid>" or "rsk_test_<uuid>", so the prefix is IDENTICAL for every
-- live (or test) key. The prefix "lookup" therefore returned EVERY active key
-- and bcrypt-compared each one on every request — O(number of keys) bcrypt ops
-- per auth (a serious latency/DoS problem as tenants grow).
--
-- key_lookup is safe to index and match directly because API keys are
-- high-entropy random tokens (a UUIDv4 suffix), not low-entropy passwords: a
-- SHA-256 of a 122-bit random value is preimage-resistant, and auth still
-- bcrypt-verifies the single matched row for defense in depth.
--
-- Nullable + partial unique index: existing keys have key_lookup = NULL until
-- the app backfills them on first successful auth (their plaintext is not
-- stored, so they cannot be backfilled by SQL alone).
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_lookup CHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS uq_api_keys_key_lookup ON api_keys (key_lookup) WHERE key_lookup IS NOT NULL;
