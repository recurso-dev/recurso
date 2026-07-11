package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type UsageRepository interface {
	RecordEvent(ctx context.Context, event *domain.UsageEvent) error
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
}
