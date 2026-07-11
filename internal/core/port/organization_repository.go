package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*domain.Organization, error)
	AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error
	RemoveTenant(ctx context.Context, orgID, tenantID uuid.UUID) error
	ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error)
}
