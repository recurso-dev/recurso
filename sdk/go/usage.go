package recurso

import (
	"context"
	"net/http"
	"time"
)

// UsageBucket is one date-truncated bucket of aggregated usage.
type UsageBucket struct {
	Period    time.Time `json:"period"`
	Dimension string    `json:"dimension"`
	Quantity  int64     `json:"quantity"`
}

// UsageDimension is a dimension-catalog row for the tenant.
type UsageDimension struct {
	Dimension  string    `json:"dimension"`
	EventCount int64     `json:"event_count"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

// UsageRecordParams reports a metered usage event.
type UsageRecordParams struct {
	SubscriptionID string `json:"subscription_id"`
	CustomerID     string `json:"customer_id"`
	Dimension      string `json:"dimension"`
	Quantity       int64  `json:"quantity"`
}

// UsageRecordResult is returned by Record.
type UsageRecordResult struct {
	Status  string `json:"status"`
	EventID string `json:"event_id"`
}

// UsageQueryParams filters a windowed usage query. At least one of
// SubscriptionID or CustomerID is required.
type UsageQueryParams struct {
	SubscriptionID string
	CustomerID     string
	Dimension      string
	From           string
	To             string
	Granularity    string
}

// UsageQueryResult is the response from Query: buckets plus the resolved window.
type UsageQueryResult struct {
	Data        []UsageBucket `json:"data"`
	From        time.Time     `json:"from"`
	To          time.Time     `json:"to"`
	Granularity string        `json:"granularity"`
}

// UsageService groups the metered-usage endpoints.
type UsageService struct{ client *Client }

// Record records a metered usage event against a subscription dimension.
func (s *UsageService) Record(ctx context.Context, params *UsageRecordParams) (*UsageRecordResult, error) {
	var out UsageRecordResult
	if err := s.client.do(ctx, http.MethodPost, "/usage/events", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Query aggregates usage events into time buckets over a window.
func (s *UsageService) Query(ctx context.Context, params *UsageQueryParams) (*UsageQueryResult, error) {
	path := "/usage"
	if params != nil {
		path = newQuery().
			str("subscription_id", params.SubscriptionID).
			str("customer_id", params.CustomerID).
			str("dimension", params.Dimension).
			str("from", params.From).
			str("to", params.To).
			str("granularity", params.Granularity).
			apply(path)
	}
	var out UsageQueryResult
	if err := s.client.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Dimensions returns the tenant's usage-dimension catalog.
func (s *UsageService) Dimensions(ctx context.Context) ([]UsageDimension, error) {
	return getData[[]UsageDimension](ctx, s.client, http.MethodGet, "/usage/dimensions", nil)
}
