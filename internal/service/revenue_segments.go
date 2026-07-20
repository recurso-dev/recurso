package service

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CustomerLookup resolves a customer (for the customer's billing country).
type CustomerLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	// List enables one bulk load instead of a per-subscription GetByID — the
	// N+1 that made revenue-by-geography's first (uncached) hit take seconds
	// against a remote database.
	List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
}

// SetCustomerLookup wires customer resolution, enabling revenue-by-geography.
func (s *AnalyticsService) SetCustomerLookup(l CustomerLookup) {
	s.customers = l
}

// countryNames maps common ISO-3166 alpha-2 codes to display names; anything
// else is shown as-is (or "Unknown" when the customer has no country).
var countryNames = map[string]string{
	"IN": "India", "US": "United States", "GB": "United Kingdom", "CA": "Canada",
	"AU": "Australia", "DE": "Germany", "FR": "France", "SG": "Singapore",
	"AE": "United Arab Emirates", "NL": "Netherlands", "JP": "Japan", "BR": "Brazil",
}

func countryLabel(code string) string {
	if code == "" {
		return "Unknown"
	}
	if name, ok := countryNames[code]; ok {
		return name
	}
	return code
}

// GetRevenueByGeography breaks current MRR down by the customer's billing
// country, largest-first with each country's share of total. Customers are
// resolved via the customer lookup (cached); a missing country falls into
// "Unknown". Requires SetCustomerLookup — otherwise everything is "Unknown".
func (s *AnalyticsService) GetRevenueByGeography(ctx context.Context, tenantID uuid.UUID) (*domain.RevenueByGeographyReport, error) {
	subs, err := s.subRepo.GetActiveSubscriptions(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)

	type agg struct {
		mrr   int64
		count int
	}
	planCache := make(map[uuid.UUID]*domain.Plan)
	// Bulk-load every customer's country up front: one query instead of one
	// GetByID round-trip per subscription (the page's slow first paint).
	custCountry := make(map[uuid.UUID]string) // customerID -> country code
	if s.customers != nil {
		if all, err := s.customers.List(ctx, tenantID, domain.CustomerFilter{Limit: 10000}); err == nil {
			for _, c := range all {
				custCountry[c.ID] = c.BillingAddress.Country
			}
		}
	}
	byCountry := make(map[string]*agg)
	var total int64

	for _, sub := range subs {
		plan, ok := planCache[sub.PlanID]
		if !ok {
			p, err := s.planRepo.GetByID(ctx, sub.PlanID)
			if err != nil || p == nil {
				continue
			}
			plan = p
			planCache[sub.PlanID] = plan
		}
		if len(plan.Prices) == 0 {
			continue
		}
		currency := plan.Prices[0].Currency
		if currency == "" {
			currency = reporting
		}
		monthly := monthlyMinorUnits(plan.Prices[0].Amount, plan.IntervalUnit, plan.IntervalCount)
		converted, _, err := normalizer.convert(ctx, monthly, currency, reporting)
		if err != nil {
			continue
		}

		country, seen := custCountry[sub.CustomerID]
		if !seen && s.customers != nil {
			// Fallback for customers past the bulk-load window.
			if cust, err := s.customers.GetByID(ctx, sub.CustomerID); err == nil && cust != nil {
				country = cust.BillingAddress.Country
			}
			custCountry[sub.CustomerID] = country
		}

		a := byCountry[country]
		if a == nil {
			a = &agg{}
			byCountry[country] = a
		}
		a.mrr += converted
		a.count++
		total += converted
	}

	report := &domain.RevenueByGeographyReport{
		ReportingCurrency: reporting,
		TotalMRR:          total,
		Segments:          make([]domain.RevenueSegment, 0, len(byCountry)),
	}
	for code, a := range byCountry {
		share := 0.0
		if total > 0 {
			share = float64(a.mrr) / float64(total) * 100
		}
		report.Segments = append(report.Segments, domain.RevenueSegment{
			Key:           code,
			Label:         countryLabel(code),
			MRR:           a.mrr,
			Subscriptions: a.count,
			SharePct:      share,
		})
	}
	sort.Slice(report.Segments, func(i, j int) bool {
		if report.Segments[i].MRR != report.Segments[j].MRR {
			return report.Segments[i].MRR > report.Segments[j].MRR
		}
		return report.Segments[i].Label < report.Segments[j].Label
	})
	return report, nil
}

// GetRevenueByPlan breaks current MRR down by plan: for each active
// subscription, its monthly-normalized price is converted to the reporting
// currency and summed per plan. Segments are returned largest-first with each
// plan's share of total MRR.
func (s *AnalyticsService) GetRevenueByPlan(ctx context.Context, tenantID uuid.UUID) (*domain.RevenueByPlanReport, error) {
	subs, err := s.subRepo.GetActiveSubscriptions(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)

	type agg struct {
		label string
		mrr   int64
		count int
	}
	planCache := make(map[uuid.UUID]*domain.Plan)
	byPlan := make(map[uuid.UUID]*agg)
	var total int64

	for _, sub := range subs {
		plan, ok := planCache[sub.PlanID]
		if !ok {
			p, err := s.planRepo.GetByID(ctx, sub.PlanID)
			if err != nil || p == nil {
				continue
			}
			plan = p
			planCache[sub.PlanID] = plan
		}
		if len(plan.Prices) == 0 {
			continue
		}
		currency := plan.Prices[0].Currency
		if currency == "" {
			currency = reporting
		}
		monthly := monthlyMinorUnits(plan.Prices[0].Amount, plan.IntervalUnit, plan.IntervalCount)
		converted, _, err := normalizer.convert(ctx, monthly, currency, reporting)
		if err != nil {
			continue // unconvertible currency excluded from the normalized total
		}
		a := byPlan[sub.PlanID]
		if a == nil {
			label := plan.Name
			if label == "" {
				label = plan.Code
			}
			a = &agg{label: label}
			byPlan[sub.PlanID] = a
		}
		a.mrr += converted
		a.count++
		total += converted
	}

	report := &domain.RevenueByPlanReport{
		ReportingCurrency: reporting,
		TotalMRR:          total,
		Segments:          make([]domain.RevenueSegment, 0, len(byPlan)),
	}
	for id, a := range byPlan {
		share := 0.0
		if total > 0 {
			share = float64(a.mrr) / float64(total) * 100
		}
		report.Segments = append(report.Segments, domain.RevenueSegment{
			Key:           id.String(),
			Label:         a.label,
			MRR:           a.mrr,
			Subscriptions: a.count,
			SharePct:      share,
		})
	}
	sort.Slice(report.Segments, func(i, j int) bool {
		if report.Segments[i].MRR != report.Segments[j].MRR {
			return report.Segments[i].MRR > report.Segments[j].MRR
		}
		return report.Segments[i].Label < report.Segments[j].Label
	})
	return report, nil
}
