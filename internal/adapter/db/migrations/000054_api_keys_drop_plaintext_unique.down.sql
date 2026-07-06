-- Restoring the constraint requires deduplicating empty key_values first;
-- intentionally a no-op guard: only add back if key_value is repopulated.
-- ALTER TABLE api_keys ADD CONSTRAINT api_keys_key_value_key UNIQUE (key_value);
SELECT 1;
