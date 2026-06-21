package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type UsageRepository interface {
	RecordEvent(ctx context.Context, event *domain.UsageEvent) error
	GetUsageForPeriod(ctx context.Context, subID string, dimension string, start, end time.Time) (int64, error)
	GetUsageStats(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageStats, error)
}
