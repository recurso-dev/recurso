package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type CancelFlowRepository interface {
	// Flow CRUD
	CreateFlow(ctx context.Context, flow *domain.CancelFlow) error
	GetFlowByID(ctx context.Context, id uuid.UUID) (*domain.CancelFlow, error)
	GetDefaultFlowForTenant(ctx context.Context, tenantID uuid.UUID) (*domain.CancelFlow, error)
	ListFlowsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.CancelFlow, error)
	UpdateFlow(ctx context.Context, flow *domain.CancelFlow) error

	// Step CRUD
	CreateStep(ctx context.Context, step *domain.CancelFlowStep) error
	GetStepsByFlow(ctx context.Context, flowID uuid.UUID) ([]domain.CancelFlowStep, error)
	UpdateStep(ctx context.Context, step *domain.CancelFlowStep, tenantID uuid.UUID) error
	DeleteStep(ctx context.Context, id, tenantID uuid.UUID) error

	// Session CRUD
	CreateSession(ctx context.Context, session *domain.CancelFlowSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.CancelFlowSession, error)
	UpdateSession(ctx context.Context, session *domain.CancelFlowSession) error
	GetRecentSessionByCustomer(ctx context.Context, customerID uuid.UUID) (*domain.CancelFlowSession, error)

	// Analytics
	GetSessionStats(ctx context.Context, tenantID uuid.UUID, flowID uuid.UUID) (*domain.FlowStats, error)
}
