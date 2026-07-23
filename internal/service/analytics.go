package service

import (
	"context"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// avgDaysPerMonth is the Gregorian average (365.25 / 12), used to normalize
// day- and week-billed plans into a monthly-equivalent figure.
const avgDaysPerMonth = 365.25 / 12

// monthlyMinorUnits normalizes a plan's list price — charged once per
// (IntervalCount × IntervalUnit) — into a per-month figure in the same minor
// units, so MRR sums correctly across monthly, annual, quarterly, weekly and
// daily plans. An unset or unrecognized interval is treated as monthly, which
// preserves the engine's prior behavior for plans that never set an interval.
func monthlyMinorUnits(amount int64, unit domain.IntervalUnit, count int) int64 {
	if count <= 0 {
		count = 1
	}
	var periodMonths float64
	switch unit {
	case domain.IntervalYear:
		periodMonths = 12 * float64(count)
	case domain.IntervalWeek:
		periodMonths = float64(count) * 7 / avgDaysPerMonth
	case domain.IntervalDay:
		periodMonths = float64(count) / avgDaysPerMonth
	default: // IntervalMonth, "" (unset), or anything unknown → month-like
		periodMonths = float64(count)
	}
	if periodMonths <= 0 {
		return amount
	}
	return int64(math.Round(float64(amount) / periodMonths))
}

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

	snapshots  MRRSnapshotStore      // optional; enables MRR snapshot capture + waterfall
	agingStore InvoiceAgingStore     // optional; enables invoice aging
	customers  CustomerLookup        // optional; enables revenue-by-geography
	entities   analyticsEntityReader // optional; enables per-entity MRR scoping
}

// analyticsEntityReader resolves legal entities so MRR can be reported per entity
// (Multi-Entity Books). Optional/nil-safe: without it, snapshots carry the
// subscription's raw entity (primary → nil) and per-entity filtering of the
// live GetMRR on the primary entity is unavailable — consolidated still works.
type analyticsEntityReader interface {
	GetPrimary(ctx context.Context, tenantID uuid.UUID) (*domain.Entity, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error)
}

// SetEntityReader wires per-entity MRR scoping.
func (s *AnalyticsService) SetEntityReader(r analyticsEntityReader) { s.entities = r }

// primaryEntityID returns the tenant's primary entity id, or nil when no reader
// is wired. Used to give primary-entity subscriptions (whose entity_id is NULL)
// a concrete id for reporting.
func (s *AnalyticsService) primaryEntityID(ctx context.Context, tenantID uuid.UUID) *uuid.UUID {
	if s.entities == nil {
		return nil
	}
	if e, err := s.entities.GetPrimary(ctx, tenantID); err == nil && e != nil {
		id := e.ID
		return &id
	}
	return nil
}

// effectiveEntityID is a subscription's concrete reporting entity — its own
// entity_id, or the primary's id when it's on the primary (subEntityID nil).
func effectiveEntityID(subEntityID, primaryID *uuid.UUID) *uuid.UUID {
	if subEntityID != nil {
		return subEntityID
	}
	return primaryID
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

// SetSnapshotStore wires the per-subscription MRR history used by the daily
// capture job and the MRR waterfall.
func (s *AnalyticsService) SetSnapshotStore(store MRRSnapshotStore) {
	s.snapshots = store
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
	ARR               int64                  `json:"arr"` // annual run-rate = normalized MRR × 12
	ReportingCurrency string                 `json:"reporting_currency"`
	Breakdown         []MRRCurrencyBreakdown `json:"breakdown"`
	FX                *FXSnapshot            `json:"fx,omitempty"`
}

// GetMRR calculates Monthly Recurring Revenue, normalized to the tenant's
// reporting currency (tenant.BaseCurrency, else the configured default).
// Simplification P3: Sum of all Active Subscriptions * Plan Amount (normalized to Monthly).
// GetMRR takes an optional entityID: nil = all entities (consolidated, the
// historical behavior); a concrete entity id scopes MRR to that legal entity.
func (s *AnalyticsService) GetMRR(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) (*MRRMetrics, error) {
	subs, err := s.subRepo.GetActiveSubscriptions(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	reporting := s.resolveReportingCurrency(ctx, tenantID)

	// For a per-entity view, resolve the primary once so primary-entity subs
	// (entity_id NULL) match their concrete entity id.
	var primaryID *uuid.UUID
	if entityID != nil {
		primaryID = s.primaryEntityID(ctx, tenantID)
	}

	// Sum MRR per currency. Plan lookups are cached to avoid repeated queries.
	planCache := make(map[uuid.UUID]*domain.Plan)
	perCurrency := make(map[string]int64)
	perCurrencyCount := make(map[string]int)
	for _, sub := range subs {
		if entityID != nil {
			eff := effectiveEntityID(sub.EntityID, primaryID)
			if eff == nil || *eff != *entityID {
				continue
			}
		}
		plan, ok := planCache[sub.PlanID]
		if !ok {
			p, err := s.planRepo.GetByID(ctx, sub.PlanID)
			// GetByID returns (nil, nil) for a not-found plan, so guard on nil
			// too — otherwise the len(plan.Prices) below nil-derefs and 500s.
			if err != nil || p == nil {
				continue
			}
			plan = p
			planCache[sub.PlanID] = plan
		}

		// Normalize the plan's list price to a monthly-equivalent figure so
		// annual/quarterly/weekly plans contribute the right MRR (an annual plan
		// priced 12000/yr is 1000/mo, not 12000).
		if len(plan.Prices) > 0 {
			currency := plan.Prices[0].Currency
			if currency == "" {
				currency = reporting
			}
			perCurrency[currency] += monthlyMinorUnits(plan.Prices[0].Amount, plan.IntervalUnit, plan.IntervalCount)
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
		ARR:               normalized * 12,
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
