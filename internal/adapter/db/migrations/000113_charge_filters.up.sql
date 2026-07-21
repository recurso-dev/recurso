-- A4: dimensional pricing via charge filters.
--
-- A charge may key on ONE event property (filter_key) and price distinct values
-- of that property differently: filters is a JSON array of {value, amounts}.
-- At rating time each configured value bills its matching events at its own
-- amounts (one line each); events matching no value fall through to the
-- charge's base `amounts`. Filter-less charges (filter_key = '') are unchanged.

ALTER TABLE plan_charges
    ADD COLUMN IF NOT EXISTS filter_key TEXT NOT NULL DEFAULT '';

ALTER TABLE plan_charges
    ADD COLUMN IF NOT EXISTS filters JSONB NOT NULL DEFAULT '[]'::jsonb;
