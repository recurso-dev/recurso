package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CloudBillingService implements "Recurso runs on Recurso": it mirrors every
// signup tenant as a Customer inside the platform (founder) tenant's own
// Recurso account, so the founder bills their cloud tenants with the same
// engine every tenant uses on their own customers.
//
// Increment 1 is deliberately money-free: it creates the customer link only.
// The Recurso Cloud plan, usage metering, and charging land in later
// increments — nothing here posts a ledger leg or moves money.
type CloudBillingService struct {
	platformTenantID uuid.UUID
	customers        cloudCustomerCreator
	repo             cloudMappingRepo
	tenants          cloudTenantLister
	logger           *slog.Logger
}

// The narrow interfaces the service depends on — accepted (rather than the
// concrete *CustomerService / repos) so the logic is unit-testable with fakes
// and no database.
type cloudCustomerCreator interface {
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error)
}

type cloudMappingRepo interface {
	GetByTenant(ctx context.Context, platformTenantID, tenantID uuid.UUID) (*domain.CloudTenantCustomer, error)
	Create(ctx context.Context, m *domain.CloudTenantCustomer) error
}

type cloudTenantLister interface {
	ListTenants(ctx context.Context) ([]*domain.Tenant, error)
}

func NewCloudBillingService(
	platformTenantID uuid.UUID,
	customers cloudCustomerCreator,
	repo cloudMappingRepo,
	tenants cloudTenantLister,
	logger *slog.Logger,
) *CloudBillingService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloudBillingService{
		platformTenantID: platformTenantID,
		customers:        customers,
		repo:             repo,
		tenants:          tenants,
		logger:           logger,
	}
}

// ProvisionTenant idempotently mirrors one signup tenant as a Customer inside
// the platform tenant's account. It never mirrors the platform tenant into
// itself (you don't bill yourself), and is a no-op when the mapping already
// exists — so a retried signup or a re-run backfill can't create duplicates.
func (s *CloudBillingService) ProvisionTenant(ctx context.Context, tenant *domain.Tenant) error {
	if tenant == nil {
		return nil
	}
	if tenant.ID == s.platformTenantID {
		return nil // the founder's own workspace is not a customer of itself
	}

	existing, err := s.repo.GetByTenant(ctx, s.platformTenantID, tenant.ID)
	if err != nil {
		return fmt.Errorf("check cloud mapping: %w", err)
	}
	if existing != nil {
		return nil // already provisioned
	}

	cust, err := s.customers.CreateCustomer(ctx, CreateCustomerInput{
		TenantID: s.platformTenantID,
		Email:    tenant.Email,
		Name:     tenant.Name,
	})
	if err != nil {
		return fmt.Errorf("create cloud customer: %w", err)
	}

	if err := s.repo.Create(ctx, &domain.CloudTenantCustomer{
		ID:               uuid.New(),
		PlatformTenantID: s.platformTenantID,
		TenantID:         tenant.ID,
		CustomerID:       cust.ID,
		CreatedAt:        time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("record cloud mapping: %w", err)
	}

	s.logger.Info("recurso cloud: provisioned customer for tenant",
		"tenant_id", tenant.ID, "customer_id", cust.ID)
	return nil
}

// Backfill provisions a cloud customer for every existing tenant that does not
// have one yet. It is idempotent and safe to run on every boot; it returns the
// number of tenants newly provisioned. A per-tenant failure is logged and
// skipped so one bad row can't stall the whole sweep.
func (s *CloudBillingService) Backfill(ctx context.Context) (int, error) {
	tenants, err := s.tenants.ListTenants(ctx)
	if err != nil {
		return 0, fmt.Errorf("list tenants: %w", err)
	}
	provisioned := 0
	for _, t := range tenants {
		if t.ID == s.platformTenantID {
			continue
		}
		existing, err := s.repo.GetByTenant(ctx, s.platformTenantID, t.ID)
		if err != nil {
			s.logger.Warn("recurso cloud backfill: mapping check failed", "tenant_id", t.ID, "error", err)
			continue
		}
		if existing != nil {
			continue
		}
		if err := s.ProvisionTenant(ctx, t); err != nil {
			s.logger.Warn("recurso cloud backfill: provision failed", "tenant_id", t.ID, "error", err)
			continue
		}
		provisioned++
	}
	return provisioned, nil
}
