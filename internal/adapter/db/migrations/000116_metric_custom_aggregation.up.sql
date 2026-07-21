-- A2: the `custom` aggregation. A metric evaluates a sandboxed per-event
-- expression (e.g. "quantity * properties.multiplier") and sums the results
-- into the period quantity. The expression lives in a new column; field_name
-- stays empty for custom (it uses expression, not a property name).
--
-- Widens both constraints (building on 000115, which added latest/percentile):
--   * aggregation_type gains 'custom';
--   * a new field_check variant requires expression to be set iff the type is
--     custom, and empty otherwise, while preserving the unique/percentile
--     field_name rules.

ALTER TABLE billable_metrics
    ADD COLUMN IF NOT EXISTS expression TEXT NOT NULL DEFAULT '';

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique', 'latest', 'percentile', 'custom'));

-- field_name: set only for unique/percentile. expression: set only for custom.
-- The two are mutually exclusive and each is empty for the aggregations that
-- don't use it.
ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type IN ('unique', 'percentile') AND field_name <> '' AND expression = '') OR
        (aggregation_type = 'custom' AND field_name = '' AND expression <> '') OR
        (aggregation_type NOT IN ('unique', 'percentile', 'custom') AND field_name = '' AND expression = '')
    );
