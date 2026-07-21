-- Revert weighted_sum. Fails if any weighted_sum metric rows exist — delete them
-- first.
ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique', 'latest', 'percentile', 'custom'));

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type IN ('unique', 'percentile') AND field_name <> '' AND expression = '') OR
        (aggregation_type = 'custom' AND field_name = '' AND expression <> '') OR
        (aggregation_type NOT IN ('unique', 'percentile', 'custom') AND field_name = '' AND expression = '')
    );
