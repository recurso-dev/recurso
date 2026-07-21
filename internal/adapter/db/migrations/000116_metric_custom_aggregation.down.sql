-- Revert the custom aggregation. Fails if any custom metric rows exist — delete
-- them first.
ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique', 'latest', 'percentile'));

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type IN ('unique', 'percentile') AND field_name <> '') OR
        (aggregation_type NOT IN ('unique', 'percentile') AND field_name = '')
    );

ALTER TABLE billable_metrics
    DROP COLUMN IF EXISTS expression;
