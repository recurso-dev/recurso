-- Fix: the `latest` and `percentile` aggregations shipped (domain + service +
-- rating support) but their DB constraints were never widened, so creating a
-- metric with either type is rejected at INSERT time in production:
--   * aggregation_type's CHECK still only admits count/sum/max/unique;
--   * field_check forces field_name='' for every non-unique type, but a
--     percentile metric stores its percentile (e.g. "95") in field_name.
-- This migration widens both constraints so the two aggregations are usable.
-- (The `custom` expression aggregation adds itself to these lists in a later
-- migration.)

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique', 'latest', 'percentile'));

-- field_name carries data only for `unique` (the property to count DISTINCT) and
-- `percentile` (the percentile 1-99). Every other type must leave it empty.
ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type IN ('unique', 'percentile') AND field_name <> '') OR
        (aggregation_type NOT IN ('unique', 'percentile') AND field_name = '')
    );
