package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// Add to existing file or create new
type CustomerRepository interface {
	Create(ctx context.Context, customer *domain.Customer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	GetByReferralCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Customer, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
	// FindByEmailAcrossTenants is cross-tenant by design: portal login
	// identifies customers by email before any tenant is known. Never call
	// it from tenant-scoped request paths.
	FindByEmailAcrossTenants(ctx context.Context, email string) ([]*domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) error
	UpdateRisk(ctx context.Context, customerID uuid.UUID, score int, factors map[string]interface{}) error
	UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error
}
