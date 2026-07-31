-- Distinguish auto-established economic nexus (written by the nightly
-- economic-nexus scheduler) from nexus a tenant MANUALLY declared. The US
-- sales-tax collection gate treats "the tenant has declared nexus" as
-- "collect only in the listed states"; auto-establishing economic nexus into
-- the same set flipped a tenant who was deferring to a provider account into
-- that restrictive mode, silently halting collection in states with real
-- (provider) physical nexus not mirrored here. The gate now triggers only on
-- MANUAL declarations (auto_established = false); economic states are still
-- collected (they make in_state true), but they no longer take over nexus
-- management from the provider.
ALTER TABLE tenant_tax_nexus
    ADD COLUMN auto_established BOOLEAN NOT NULL DEFAULT false;

-- Backfill: existing economic rows were written by the scheduler (economic
-- nexus is threshold-triggered, not something tenants declare by hand), so
-- mark them auto-established to un-poison already-affected tenants.
UPDATE tenant_tax_nexus SET auto_established = true WHERE nexus_type = 'economic';
