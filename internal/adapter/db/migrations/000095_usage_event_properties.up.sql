-- Usage-based billing v1 (spec_usage_billing.md, D2): free-form event
-- properties. Needed by the 'unique' aggregation (count distinct
-- properties->>field_name) and the foundation for charge filters/group-by.
-- NULL for property-less events; existing ingest paths are unaffected.

ALTER TABLE usage_events ADD COLUMN IF NOT EXISTS properties JSONB;
