package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// EntitlementRepository persists plan-level feature grants.
type EntitlementRepository interface {
	// ReplaceForPlan atomically replaces the plan's full entitlement set:
	// rows absent from ents are deleted (PUT semantics).
	ReplaceForPlan(ctx context.Context, tenantID, planID uuid.UUID, ents []domain.Entitlement) error
	ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Entitlement, error)
	ListByPlanIDs(ctx context.Context, tenantID uuid.UUID, planIDs []uuid.UUID) ([]domain.Entitlement, error)
	// CheckFeature answers "does this customer have this feature?" in a
	// single indexed query over the customer's active/trialing
	// subscriptions (hot path — no N+1).
	CheckFeature(ctx context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error)
}
