package domain

import (
	"time"

	"github.com/google/uuid"
)

// Usage threshold alerts (Lago-parity B3). An alert watches one metric on
// one subscription and fires AT MOST ONCE per billing period per threshold
// when the period's aggregated usage crosses it — via the webhook event
// EventUsageAlertTriggered plus an email notification.

// UsageAlertThresholdType selects how Threshold is interpreted.
type UsageAlertThresholdType string

const (
	// AlertThresholdQuantity fires when the period's aggregated quantity
	// reaches an absolute value.
	AlertThresholdQuantity UsageAlertThresholdType = "quantity"
	// AlertThresholdPercentOfLimit fires when usage reaches a percentage of
	// the customer's entitlement limit whose feature_key equals the metric
	// code (e.g. 80 = 80% of the plan's included quota).
	AlertThresholdPercentOfLimit UsageAlertThresholdType = "percent_of_limit"
)

// EventUsageAlertTriggered is the webhook event type an alert emits.
const EventUsageAlertTriggered = "usage.alert.triggered"

// UsageAlert is one configured threshold.
type UsageAlert struct {
	ID             uuid.UUID               `json:"id"`
	TenantID       uuid.UUID               `json:"tenant_id"`
	SubscriptionID uuid.UUID               `json:"subscription_id"`
	MetricCode     string                  `json:"metric_code"`
	ThresholdType  UsageAlertThresholdType `json:"threshold_type"`
	// Threshold is an absolute quantity, or a percentage (1-1000) for
	// percent_of_limit (values above 100 alert on overage).
	Threshold int64 `json:"threshold"`
	// LastFiredPeriodStart dedups firing: one alert per billing period.
	LastFiredPeriodStart *time.Time `json:"last_fired_period_start,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
