package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// GetUnitEconomics computes ARPA, ARPU and LTV from live MRR and active
// subscriptions. LTV = ARPA / monthly revenue-churn, where churn comes from the
// trailing-30-day MRR waterfall — so it's only reported once snapshot history
// exists and churn is non-zero (HasLTV).
func (s *AnalyticsService) GetUnitEconomics(ctx context.Context, tenantID uuid.UUID) (*domain.UnitEconomics, error) {
	mrrM, err := s.GetMRR(ctx, tenantID, nil) // tenant-wide (all entities)
	if err != nil {
		return nil, err
	}
	subs, err := s.subRepo.GetActiveSubscriptions(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	customers := make(map[uuid.UUID]struct{}, len(subs))
	for _, sub := range subs {
		customers[sub.CustomerID] = struct{}{}
	}

	ue := &domain.UnitEconomics{
		ReportingCurrency:   mrrM.ReportingCurrency,
		MRR:                 mrrM.NormalizedMRR,
		ActiveCustomers:     len(customers),
		ActiveSubscriptions: len(subs),
	}
	if ue.ActiveCustomers > 0 {
		ue.ARPA = int64(math.Round(float64(ue.MRR) / float64(ue.ActiveCustomers)))
	}
	if ue.ActiveSubscriptions > 0 {
		ue.ARPU = int64(math.Round(float64(ue.MRR) / float64(ue.ActiveSubscriptions)))
	}

	// LTV needs a churn rate, which needs MRR history. Derive it from the
	// trailing-30-day waterfall's revenue churn when that history exists.
	if s.snapshots != nil {
		end := time.Now()
		wf, err := s.GetMRRWaterfall(ctx, tenantID, nil, end.AddDate(0, 0, -30), end)
		if err == nil && wf.HasStartHistory && wf.StartingMRR > 0 && wf.Churned > 0 {
			churnRate := float64(wf.Churned) / float64(wf.StartingMRR)
			ue.MonthlyChurnRate = churnRate * 100
			ue.LTV = int64(math.Round(float64(ue.ARPA) / churnRate))
			ue.HasLTV = true
		}
	}
	return ue, nil
}
