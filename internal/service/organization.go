package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type OrganizationService struct {
	orgRepo     port.OrganizationRepository
	subRepo     port.SubscriptionRepository
	planRepo    port.PlanRepository
}

func NewOrganizationService(
	orgRepo port.OrganizationRepository,
	subRepo port.SubscriptionRepository,
	planRepo port.PlanRepository,
) *OrganizationService {
	return &OrganizationService{
		orgRepo:  orgRepo,
		subRepo:  subRepo,
		planRepo: planRepo,
	}
}

func (s *OrganizationService) Create(ctx context.Context, name, ownerEmail string) (*domain.Organization, error) {
	org := &domain.Organization{
		ID:         uuid.New(),
		Name:       name,
		OwnerEmail: ownerEmail,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return s.orgRepo.GetByID(ctx, id)
}

func (s *OrganizationService) AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	// Verify org exists
	if _, err := s.orgRepo.GetByID(ctx, orgID); err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	return s.orgRepo.AddTenant(ctx, orgID, tenantID)
}

func (s *OrganizationService) ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error) {
	return s.orgRepo.ListTenants(ctx, orgID)
}

type OrgMRRMetrics struct {
	TotalMRR int64            `json:"total_mrr"`
	Currency string           `json:"currency"`
	ByTenant []TenantMRR      `json:"by_tenant"`
}

type TenantMRR struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	MRR        int64     `json:"mrr"`
}

func (s *OrganizationService) GetConsolidatedMRR(ctx context.Context, orgID uuid.UUID) (*OrgMRRMetrics, error) {
	tenants, err := s.orgRepo.ListTenants(ctx, orgID)
	if err != nil {
		return nil, err
	}

	metrics := &OrgMRRMetrics{
		Currency: "USD",
		ByTenant: make([]TenantMRR, 0),
	}

	// Cache plan lookups to avoid repeated queries for the same plan
	planCache := make(map[uuid.UUID]*domain.Plan)

	for _, tenant := range tenants {
		subs, err := s.subRepo.List(ctx, tenant.ID, domain.SubscriptionFilter{Status: "active", Limit: 1000})
		if err != nil {
			continue
		}

		var tenantMRR int64
		for _, sub := range subs {
			plan, ok := planCache[sub.PlanID]
			if !ok {
				p, err := s.planRepo.GetByID(ctx, sub.PlanID)
				if err != nil {
					continue
				}
				plan = p
				planCache[sub.PlanID] = plan
			}
			if len(plan.Prices) > 0 {
				tenantMRR += plan.Prices[0].Amount
			}
		}

		metrics.ByTenant = append(metrics.ByTenant, TenantMRR{
			TenantID:   tenant.ID,
			TenantName: tenant.Name,
			MRR:        tenantMRR,
		})
		metrics.TotalMRR += tenantMRR
	}

	return metrics, nil
}
