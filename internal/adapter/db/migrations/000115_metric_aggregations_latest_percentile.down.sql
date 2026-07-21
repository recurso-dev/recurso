-- Revert to the original (pre-latest/percentile) constraints. This will fail if
-- any latest/percentile metric rows exist — delete them first.
ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique'));

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type = 'unique' AND field_name <> '') OR
        (aggregation_type <> 'unique' AND field_name = '')
    );
