-- US economic-nexus threshold alerting (Track D · D1). The nexus scheduler
-- already establishes economic nexus on a crossing, but silently — so a
-- registration obligation can pass unnoticed. This table dedups the proactive
-- alert: at most one per (tenant, state, calendar year, level).

CREATE TABLE IF NOT EXISTS nexus_alerts (
    id            UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    state_code    TEXT NOT NULL,
    year          INT  NOT NULL,
    -- 'approaching' — reached the approaching band (default 80%) of the state's
    -- economic-nexus threshold; 'crossed' — threshold crossed, nexus established.
    level         TEXT NOT NULL CHECK (level IN ('approaching','crossed')),
    proximity_pct INT  NOT NULL,
    sent_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- The atomic insert on this key is the dedup primitive (robust even when the
    -- scheduler lock is a no-op without Redis): a state that stays crossed is not
    -- re-alerted every day, and a new calendar year resets the picture.
    UNIQUE (tenant_id, state_code, year, level)
);

CREATE INDEX IF NOT EXISTS idx_nexus_alerts_tenant ON nexus_alerts (tenant_id);
