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
	// GetByCode resolves a metric by its code (== event dimension);
	// (nil, nil) when absent.
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.BillableMetric, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.BillableMetric, error)
	Update(ctx context.Context, m *domain.BillableMetric) error
	// Delete removes a metric. Metrics referenced by a plan charge cannot be
	// deleted (FK restricts); callers surface that as a conflict.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// UsageAlertRepository persists usage threshold alerts (Lago-parity B3).
type UsageAlertRepository interface {
	Create(ctx context.Context, a *domain.UsageAlert) error
	ListBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]domain.UsageAlert, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.UsageAlert, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// ListAll returns alerts across tenants for the sweep.
	ListAll(ctx context.Context, limit int) ([]domain.UsageAlert, error)
	// MarkFired claims the (alert, period) firing: returns true when THIS
	// call moved last_fired_period_start to periodStart, false when another
	// sweep already fired the alert for that period.
	MarkFired(ctx context.Context, id uuid.UUID, periodStart time.Time) (bool, error)
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

// AuditLogRepository persists the append-only audit trail (Lago-parity C2).
type AuditLogRepository interface {
	Insert(ctx context.Context, a *domain.AuditLog) error
	List(ctx context.Context, tenantID uuid.UUID, filter domain.AuditLogFilter) ([]domain.AuditLog, error)
}
