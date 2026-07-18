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
}
