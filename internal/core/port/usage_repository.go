package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type UsageRepository interface {
	RecordEvent(ctx context.Context, event *domain.UsageEvent) error
	// RecordEventIdempotent inserts the event, collapsing duplicates by
	// (subscription_id, transaction_id) to the original: duplicate=true and
	// the event's ID rewritten to the original's (Lago-parity C1).
	RecordEventIdempotent(ctx context.Context, event *domain.UsageEvent) (duplicate bool, err error)
	GetUsageForPeriod(ctx context.Context, subID string, dimension string, start, end time.Time) (int64, error)
	GetUsageStats(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageStats, error)

	// QueryUsage aggregates usage into date_trunc'd time buckets. The
	// filter's From/To/Granularity must already be validated/defaulted by
	// the caller (see service.UsageService.QueryUsage).
	QueryUsage(ctx context.Context, tenantID uuid.UUID, filter domain.UsageQueryFilter) ([]domain.UsageBucket, error)

	// ListRecentEvents returns the tenant's newest raw events (stream
	// debugging), optionally filtered by customer and dimension.
	ListRecentEvents(ctx context.Context, tenantID uuid.UUID, customerID *uuid.UUID, dimension string, limit, offset int) ([]domain.UsageEvent, error)

	// GetSubscriptionUsageByDimension returns, per dimension, the total
	// quantity inside [periodStart, periodEnd) alongside the lifetime total.
	GetSubscriptionUsageByDimension(ctx context.Context, tenantID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) ([]domain.SubscriptionDimensionUsage, error)

	// ListDimensions returns the tenant's distinct usage dimensions with
	// event counts and first/last seen timestamps.
	ListDimensions(ctx context.Context, tenantID uuid.UUID) ([]domain.UsageDimension, error)

	// AggregateForMetric reduces a subscription's events for the metric's
	// dimension inside [start, end) to one quantity per the metric's
	// aggregation type (count | sum | max | unique on properties->>field_name).
	// Zero events aggregate to 0 (usage-based billing v1).
	AggregateForMetric(ctx context.Context, subscriptionID uuid.UUID, metric domain.BillableMetric, start, end time.Time) (int64, error)

	// SumDynamicAmount sums the per-event dynamic_amount (minor units) for the
	// dimension inside [start, end) — the quantity a `dynamic` charge bills.
	// Zero events sum to 0.
	SumDynamicAmount(ctx context.Context, subscriptionID uuid.UUID, dimension string, start, end time.Time) (int64, error)

	// AggregateForMetricFiltered aggregates like AggregateForMetric but only
	// over events whose property propKey is IN propValues (exclude=false) or is
	// NULL / NOT IN propValues (exclude=true) — dimensional pricing (A4).
	AggregateForMetricFiltered(ctx context.Context, subscriptionID uuid.UUID, metric domain.BillableMetric, propKey string, propValues []string, exclude bool, start, end time.Time) (int64, error)

	// StreamEventsForMetric invokes fn for each event of the dimension inside
	// [start, end), in occurrence order, passing the event quantity, its
	// timestamp, and its raw string properties (nil when none). It streams rather
	// than materializing, so an aggregation folds a large period without loading
	// every event. The timestamp lets time-dependent folds (weighted_sum) weight
	// each interval; the custom aggregation ignores it. fn returning an error
	// stops iteration and is returned.
	StreamEventsForMetric(ctx context.Context, subscriptionID uuid.UUID, dimension string, start, end time.Time, fn func(quantity int64, ts time.Time, props map[string]string) error) error

	// SumQuantityBefore returns Σ quantity of the dimension's events strictly
	// before `before` — the carry-forward starting level for weighted_sum (a
	// resource provisioned in an earlier period is still active at this period's
	// start). Zero events sum to 0.
	SumQuantityBefore(ctx context.Context, subscriptionID uuid.UUID, dimension string, before time.Time) (int64, error)
}
