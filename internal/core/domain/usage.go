package domain

import (
	"time"

	"github.com/google/uuid"
)

type UsageEvent struct {
	ID             uuid.UUID `json:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	Dimension      string    `json:"dimension"` // e.g., "api_calls", "storage_gb"
	Quantity       int64     `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	// Properties are optional free-form event attributes (JSONB). The
	// unique aggregation counts distinct values of one property; they also
	// ground future charge filters/group-by. Nil for property-less events.
	Properties map[string]string `json:"properties,omitempty"`
}

// Usage query granularities for time-windowed aggregation.
const (
	UsageGranularityDay   = "day"
	UsageGranularityMonth = "month"
)

// UsageQueryFilter narrows a time-windowed usage aggregation.
//
// Tenant scoping note: usage_events has no tenant_id column, so all reads
// are scoped by joining subscriptions on subscription_id and filtering on
// subscriptions.tenant_id.
type UsageQueryFilter struct {
	SubscriptionID *uuid.UUID
	CustomerID     *uuid.UUID
	Dimension      string // optional; empty = all dimensions
	From           time.Time
	To             time.Time
	Granularity    string // UsageGranularityDay | UsageGranularityMonth
}

// UsageBucket is one date_trunc'd time bucket of aggregated usage.
type UsageBucket struct {
	Period    time.Time `json:"period"`
	Dimension string    `json:"dimension"`
	Quantity  int64     `json:"quantity"`
}

// UsageDimension is a dimension-catalog row: every dimension the tenant
// has ever reported, with first/last activity and event volume.
type UsageDimension struct {
	Dimension  string    `json:"dimension"`
	EventCount int64     `json:"event_count"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

// SubscriptionDimensionUsage is one dimension's usage for a subscription:
// the current billing period's total, the lifetime total, and — when an
// entitlement limit exists for a feature_key matching the dimension name —
// the customer's limit and what remains of it this period.
type SubscriptionDimensionUsage struct {
	Dimension        string `json:"dimension"`
	PeriodQuantity   int64  `json:"period_quantity"`
	LifetimeQuantity int64  `json:"lifetime_quantity"`
	// LimitValue is the customer's effective entitlement limit for the
	// feature_key equal to Dimension, or null when no limit grant exists.
	LimitValue *int64 `json:"limit_value"`
	// Remaining is LimitValue - PeriodQuantity (may be negative when the
	// customer is over their limit); null when LimitValue is null.
	Remaining *int64 `json:"remaining"`
}

// SubscriptionUsage is the per-subscription usage report
// (GET /v1/subscriptions/{id}/usage).
type SubscriptionUsage struct {
	SubscriptionID     uuid.UUID                    `json:"subscription_id"`
	CustomerID         uuid.UUID                    `json:"customer_id"`
	CurrentPeriodStart time.Time                    `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                    `json:"current_period_end"`
	Dimensions         []SubscriptionDimensionUsage `json:"dimensions"`
}
