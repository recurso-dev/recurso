-- Inbound webhook idempotency: records gateway webhook events we've fully
-- processed so a redelivery (gateways retry on timeout) is acknowledged without
-- re-running non-idempotent side effects (e.g. the payment-failed email, the
-- dunning bandit outcome). Recorded only AFTER successful processing, so a
-- failed delivery is still retried. (ENG-162)
CREATE TABLE IF NOT EXISTS inbound_webhook_events (
    gateway      TEXT        NOT NULL,
    event_id     TEXT        NOT NULL,
    event_type   TEXT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (gateway, event_id)
);
