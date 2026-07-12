package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type DunningCampaignRepository interface {
	// Campaign CRUD
	CreateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error
	GetCampaignByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.DunningCampaign, error)
	GetActiveCampaignForTenant(ctx context.Context, tenantID uuid.UUID, triggerEvent string) (*domain.DunningCampaign, error)
	ListCampaignsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.DunningCampaign, error)
	UpdateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error

	// Step CRUD
	CreateStep(ctx context.Context, step *domain.DunningCampaignStep) error
	GetStepsByCampaign(ctx context.Context, campaignID uuid.UUID) ([]domain.DunningCampaignStep, error)
	UpdateStep(ctx context.Context, step *domain.DunningCampaignStep) error
	DeleteStep(ctx context.Context, id uuid.UUID) error

	// Execution CRUD
	CreateExecution(ctx context.Context, exec *domain.DunningCampaignExecution) error
	GetExecutionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.DunningCampaignExecution, error)
	UpdateExecution(ctx context.Context, exec *domain.DunningCampaignExecution) error
	GetDueExecutions(ctx context.Context, now time.Time) ([]*domain.DunningCampaignExecution, error)
}
