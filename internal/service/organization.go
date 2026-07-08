package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type OrganizationService struct {
	orgRepo  port.OrganizationRepository
	subRepo  port.SubscriptionRepository
	planRepo port.PlanRepository

	fxProvider        port.ExchangeRateProvider
	fxFallback        port.ExchangeRateProvider
	reportingCurrency string
}

func NewOrganizationService(
	orgRepo port.OrganizationRepository,
	subRepo port.SubscriptionRepository,
	planRepo port.PlanRepository,
) *OrganizationService {
	return &OrganizationService{
		orgRepo:           orgRepo,
		subRepo:           subRepo,
		planRepo:          planRepo,
		reportingCurrency: "USD",
	}
}

// SetFX wires the FX provider used to normalize consolidated MRR, an optional
// static fallback, and the org-level reporting currency (REPORTING_CURRENCY
// env; organizations have no per-org currency setting yet).
func (s *OrganizationService) SetFX(provider, fallback port.ExchangeRateProvider, reportingCurrency string) {
	s.fxProvider = provider
	s.fxFallback = fallback
	if reportingCurrency != "" {
		s.reportingCurrency = reportingCurrency
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

func (s *OrganizationService) Update(ctx context.Context, orgID uuid.UUID, name, ownerEmail string) (*domain.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	if name != "" {
		org.Name = name
	}
	if ownerEmail != "" {
		org.OwnerEmail = ownerEmail
	}
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	return org, nil
}

func (s *OrganizationService) Delete(ctx context.Context, orgID uuid.UUID) error {
	if _, err := s.orgRepo.GetByID(ctx, orgID); err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}
	return s.orgRepo.Delete(ctx, orgID)
}

func (s *OrganizationService) RemoveTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	if _, err := s.orgRepo.GetByID(ctx, orgID); err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}
	return s.orgRepo.RemoveTenant(ctx, orgID, tenantID)
}

func (s *OrganizationService) List(ctx context.Context) ([]*domain.Organization, error) {
	return s.orgRepo.List(ctx)
}

type CurrencyMRR struct {
	TotalMRR int64       `json:"total_mrr"`
	Currency string      `json:"currency"`
	ByTenant []TenantMRR `json:"by_tenant"`

	// FX normalization into the reporting currency.
	ConvertedMRR int64   `json:"converted_mrr"`      // TotalMRR in the reporting currency
	Rate         float64 `json:"rate"`               // rate applied (currency -> reporting)
	FXError      string  `json:"fx_error,omitempty"` // set when conversion failed; excluded from the total
}

type OrgMRRMetrics struct {
	ByCurrency []CurrencyMRR `json:"by_currency"`

	NormalizedMRR     int64       `json:"normalized_mrr"`
	ReportingCurrency string      `json:"reporting_currency"`
	FX                *FXSnapshot `json:"fx,omitempty"`
}

type TenantMRR struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	MRR        int64     `json:"mrr"`
	Currency   string    `json:"currency"`
}

func (s *OrganizationService) GetConsolidatedMRR(ctx context.Context, orgID uuid.UUID) (*OrgMRRMetrics, error) {
	tenants, err := s.orgRepo.ListTenants(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Cache plan lookups to avoid repeated queries for the same plan
	planCache := make(map[uuid.UUID]*domain.Plan)

	// Group MRR by currency
	currencyTotals := make(map[string]int64)
	currencyTenants := make(map[string][]TenantMRR)

	for _, tenant := range tenants {
		// Consolidated MRR spans MANY tenants; the plan repo reads the tenant
		// from the context, so scope the context to THIS tenant each iteration.
		// (Using the request tenant here would fail every plan lookup and yield
		// a silently-zero MRR.)
		tctx := context.WithValue(ctx, domain.TenantIDKey, tenant.ID)

		subs, err := s.subRepo.List(tctx, tenant.ID, domain.SubscriptionFilter{Status: "active", Limit: 1000})
		if err != nil {
			continue
		}

		// Per-tenant, per-currency MRR
		tenantCurrencyMRR := make(map[string]int64)

		for _, sub := range subs {
			plan, ok := planCache[sub.PlanID]
			if !ok {
				p, err := s.planRepo.GetByID(tctx, sub.PlanID)
				if err != nil {
					continue
				}
				plan = p
				planCache[sub.PlanID] = plan
			}
			if len(plan.Prices) > 0 {
				currency := plan.Prices[0].Currency
				tenantCurrencyMRR[currency] += plan.Prices[0].Amount
			}
		}

		for currency, mrr := range tenantCurrencyMRR {
			currencyTotals[currency] += mrr
			currencyTenants[currency] = append(currencyTenants[currency], TenantMRR{
				TenantID:   tenant.ID,
				TenantName: tenant.Name,
				MRR:        mrr,
				Currency:   currency,
			})
		}
	}

	reporting := s.reportingCurrency
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)

	metrics := &OrgMRRMetrics{
		ByCurrency:        make([]CurrencyMRR, 0, len(currencyTotals)),
		ReportingCurrency: reporting,
	}
	for currency, total := range currencyTotals {
		entry := CurrencyMRR{
			TotalMRR: total,
			Currency: currency,
			ByTenant: currencyTenants[currency],
		}
		converted, rate, err := normalizer.convert(ctx, total, currency, reporting)
		if err != nil {
			entry.FXError = err.Error()
		} else {
			entry.ConvertedMRR = converted
			entry.Rate = rate
			metrics.NormalizedMRR += converted
		}
		metrics.ByCurrency = append(metrics.ByCurrency, entry)
	}
	sort.Slice(metrics.ByCurrency, func(i, j int) bool {
		return metrics.ByCurrency[i].Currency < metrics.ByCurrency[j].Currency
	})
	metrics.FX = normalizer.snapshot()

	return metrics, nil
}
