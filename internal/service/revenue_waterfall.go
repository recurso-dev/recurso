package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// summarizeWaterfall totals recognized and scheduled revenue across the monthly
// buckets. Pure so it is unit-testable without a database.
func summarizeWaterfall(tenantID uuid.UUID, buckets []domain.RevenueWaterfallBucket) *domain.RevenueWaterfall {
	var totalRecognized, totalScheduled int64
	for _, b := range buckets {
		totalRecognized += b.Recognized
		totalScheduled += b.Scheduled
	}
	return &domain.RevenueWaterfall{
		TenantID:        tenantID,
		Buckets:         buckets,
		TotalRecognized: totalRecognized,
		TotalScheduled:  totalScheduled,
	}
}

// GetWaterfall returns the tenant's recognized-plus-scheduled revenue curve,
// month by month, with totals. Read-only: the rev-rec waterfall an auditor uses
// to see how deferred revenue releases over time.
func (s *RevRecService) GetWaterfall(ctx context.Context, tenantID uuid.UUID) (*domain.RevenueWaterfall, error) {
	buckets, err := s.repo.GetWaterfall(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return summarizeWaterfall(tenantID, buckets), nil
}
