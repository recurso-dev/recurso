package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// BillableMetricRepository persists tenant-defined meters
// (usage-based billing v1, spec_usage_billing.md).
type BillableMetricRepository interface {
	Create(ctx context.Context, m *domain.BillableMetric) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BillableMetric, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.BillableMetric, error)
	Update(ctx context.Context, m *domain.BillableMetric) error
	// Delete removes a metric. Metrics referenced by a plan charge cannot be
	// deleted (FK restricts); callers surface that as a conflict.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// ChargeRepository persists plan usage charges.
type ChargeRepository interface {
	// ReplaceForPlan deletes the plan's existing charges and inserts the new
	// set in one transaction (PUT replace semantics, like entitlements).
	ReplaceForPlan(ctx context.Context, tenantID, planID uuid.UUID, charges []domain.Charge) error
	// ListByPlan returns the plan's charges with their metrics joined.
	ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Charge, error)
}

// UsageRatingRepository persists the double-billing guard rows.
type UsageRatingRepository interface {
	// Create inserts a rating claim. Returns (false, nil) when the
	// (subscription, charge, period_start) window was already rated
	// (ON CONFLICT DO NOTHING), true when this call claimed it.
	Create(ctx context.Context, r *domain.UsageRating) (bool, error)
	// Exists reports whether the window is already rated.
	Exists(ctx context.Context, subscriptionID, chargeID uuid.UUID, periodStart time.Time) (bool, error)
}
