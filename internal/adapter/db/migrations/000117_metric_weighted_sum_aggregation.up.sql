-- A2: the `weighted_sum` aggregation — a time-weighted average of a running
-- level built from per-event signed deltas (for per-time resources like seats).
-- It uses neither field_name nor expression, so it joins the "no extra config"
-- branch of field_check alongside count/sum/max/latest.

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_aggregation_type_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_aggregation_type_check
    CHECK (aggregation_type IN ('count', 'sum', 'max', 'unique', 'latest', 'percentile', 'custom', 'weighted_sum'));

ALTER TABLE billable_metrics
    DROP CONSTRAINT IF EXISTS billable_metrics_field_check;
ALTER TABLE billable_metrics
    ADD CONSTRAINT billable_metrics_field_check CHECK (
        (aggregation_type IN ('unique', 'percentile') AND field_name <> '' AND expression = '') OR
        (aggregation_type = 'custom' AND field_name = '' AND expression <> '') OR
        (aggregation_type NOT IN ('unique', 'percentile', 'custom') AND field_name = '' AND expression = '')
    );
