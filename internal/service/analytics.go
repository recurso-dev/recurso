package service

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// TenantLookup resolves a tenant so reporting can honor its base currency.
type TenantLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
}

type AnalyticsService struct {
	subRepo     port.SubscriptionRepository
	invoiceRepo port.InvoiceRepository
	planRepo    port.PlanRepository
	usageRepo   port.UsageRepository

	fxProvider        port.ExchangeRateProvider
	fxFallback        port.ExchangeRateProvider
	tenantLookup      TenantLookup
	reportingCurrency string // env-level default when the tenant has no base currency
}

func NewAnalyticsService(
	subRepo port.SubscriptionRepository,
	invoiceRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	usageRepo port.UsageRepository,
) *AnalyticsService {
	return &AnalyticsService{
		subRepo:           subRepo,
		invoiceRepo:       invoiceRepo,
		planRepo:          planRepo,
		usageRepo:         usageRepo,
		reportingCurrency: "USD",
	}
}

// SetFX wires the FX provider used to normalize multi-currency MRR, an
// optional static fallback, and the default reporting currency.
func (s *AnalyticsService) SetFX(provider, fallback port.ExchangeRateProvider, reportingCurrency string) {
	s.fxProvider = provider
	s.fxFallback = fallback
	if reportingCurrency != "" {
		s.reportingCurrency = reportingCurrency
	}
}

// SetTenantLookup enables per-tenant reporting currency (tenant.BaseCurrency).
func (s *AnalyticsService) SetTenantLookup(l TenantLookup) {
	s.tenantLookup = l
}

func (s *AnalyticsService) GetUsageStats(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageStats, error) {
	return s.usageRepo.GetUsageStats(ctx, tenantID)
}

// MRRCurrencyBreakdown is the per-currency slice of MRR before and after
// normalization into the reporting currency.
type MRRCurrencyBreakdown struct {
	Currency        string  `json:"currency"`
	Amount          int64   `json:"amount"`           // native MRR, minor units
	ConvertedAmount int64   `json:"converted_amount"` // in the reporting currency, minor units
	Rate            float64 `json:"rate"`             // rate applied (native -> reporting)
	Subscriptions   int     `json:"subscriptions"`    // active subscriptions in this currency
	Error           string  `json:"error,omitempty"`  // set when conversion failed; excluded from the total
}

type MRRMetrics struct {
	// Currency and Amount are kept for backward compatibility. Amount is now
	// the FX-normalized total (previously it was a naive cross-currency sum).
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"` // in cents

	// MRR mirrors NormalizedMRR; the frontend dashboard reads this key.
	MRR               int64                  `json:"mrr"`
	NormalizedMRR     int64                  `json:"normalized_mrr"`
	ReportingCurrency string                 `json:"reporting_currency"`
	Breakdown         []MRRCurrencyBreakdown `json:"breakdown"`
	FX                *FXSnapshot            `json:"fx,omitempty"`
}

// GetMRR calculates Monthly Recurring Revenue, normalized to the tenant's
// reporting currency (tenant.BaseCurrency, else the configured default).
// Simplification P3: Sum of all Active Subscriptions * Plan Amount (normalized to Monthly).
func (s *AnalyticsService) GetMRR(ctx context.Context, tenantID uuid.UUID) (*MRRMetrics, error) {
	subs, err := s.subRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	reporting := s.resolveReportingCurrency(ctx, tenantID)

	// Sum MRR per currency. Plan lookups are cached to avoid repeated queries.
	planCache := make(map[uuid.UUID]*domain.Plan)
	perCurrency := make(map[string]int64)
	perCurrencyCount := make(map[string]int)
	for _, sub := range subs {
		plan, ok := planCache[sub.PlanID]
		if !ok {
			p, err := s.planRepo.GetByID(ctx, sub.PlanID)
			if err != nil {
				// Skip or Log error? Skipping for robustness
				continue
			}
			plan = p
			planCache[sub.PlanID] = plan
		}

		// Simple Calc: Assuming 1 Price is Monthly.
		// If implementation requires multiple prices or yearly, we'd normalize here.
		if len(plan.Prices) > 0 {
			currency := plan.Prices[0].Currency
			if currency == "" {
				currency = reporting
			}
			perCurrency[currency] += plan.Prices[0].Amount
			perCurrencyCount[currency]++
		}
	}

	// Convert each currency bucket into the reporting currency.
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)
	breakdown := make([]MRRCurrencyBreakdown, 0, len(perCurrency))
	var normalized int64
	for currency, amount := range perCurrency {
		entry := MRRCurrencyBreakdown{
			Currency:      currency,
			Amount:        amount,
			Subscriptions: perCurrencyCount[currency],
		}
		converted, rate, err := normalizer.convert(ctx, amount, currency, reporting)
		if err != nil {
			entry.Error = err.Error()
		} else {
			entry.ConvertedAmount = converted
			entry.Rate = rate
			normalized += converted
		}
		breakdown = append(breakdown, entry)
	}
	sort.Slice(breakdown, func(i, j int) bool { return breakdown[i].Currency < breakdown[j].Currency })

	return &MRRMetrics{
		Currency:          reporting,
		Amount:            normalized,
		MRR:               normalized,
		NormalizedMRR:     normalized,
		ReportingCurrency: reporting,
		Breakdown:         breakdown,
		FX:                normalizer.snapshot(),
	}, nil
}

// resolveReportingCurrency prefers the tenant's base currency when available,
// falling back to the service-level default (REPORTING_CURRENCY env, "USD").
func (s *AnalyticsService) resolveReportingCurrency(ctx context.Context, tenantID uuid.UUID) string {
	if s.tenantLookup != nil && tenantID != uuid.Nil {
		if tenant, err := s.tenantLookup.GetByID(ctx, tenantID); err == nil && tenant != nil && tenant.BaseCurrency != "" {
			return tenant.BaseCurrency
		}
	}
	return s.reportingCurrency
}
