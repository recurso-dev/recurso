-- Opt-in anonymous telemetry (TELEMETRY_OPTIN=true). Single row holding a
-- random instance UUID (not derived from anything identifying) plus
-- check-once milestone flags so each milestone event fires at most once per
-- instance. The table stays empty unless telemetry is enabled — the row is
-- only inserted by the telemetry client. See docs/telemetry.md.
CREATE TABLE IF NOT EXISTS telemetry_instance (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    instance_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    milestone_first_plan BOOLEAN NOT NULL DEFAULT FALSE,
    milestone_first_customer BOOLEAN NOT NULL DEFAULT FALSE,
    milestone_first_invoice BOOLEAN NOT NULL DEFAULT FALSE,
    milestone_first_payment BOOLEAN NOT NULL DEFAULT FALSE
);
