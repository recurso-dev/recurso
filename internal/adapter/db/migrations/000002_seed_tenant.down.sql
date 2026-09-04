-- Intentionally a no-op. Deleting the seed tenant fails with a foreign-key
-- violation on any database that has rows under it (dev mode uses this tenant
-- id), which would leave `migrate down` dirty at version 2. 000001's down drops
-- the tenants table itself, so the seed row goes with it.
SELECT 1;
