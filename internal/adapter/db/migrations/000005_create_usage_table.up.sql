CREATE TABLE usage_events (
    id UUID PRIMARY KEY,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    dimension VARCHAR(50) NOT NULL,
    quantity BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_usage_sub_dim ON usage_events(subscription_id, dimension);
