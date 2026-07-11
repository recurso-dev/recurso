package service

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

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
