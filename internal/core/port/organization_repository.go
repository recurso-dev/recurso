package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error
	ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error)
}
