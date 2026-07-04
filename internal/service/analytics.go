package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type AnalyticsService struct {
	subRepo     port.SubscriptionRepository
	invoiceRepo port.InvoiceRepository
	planRepo    port.PlanRepository
	usageRepo   port.UsageRepository
}

func NewAnalyticsService(
	subRepo port.SubscriptionRepository, 
	invoiceRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
	usageRepo port.UsageRepository,
) *AnalyticsService {
	return &AnalyticsService{
		subRepo:     subRepo,
		invoiceRepo: invoiceRepo,
		planRepo:    planRepo,
		usageRepo:   usageRepo,
	}
}

func (s *AnalyticsService) GetUsageStats(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageStats, error) {
	return s.usageRepo.GetUsageStats(ctx, tenantID)
}

type MRRMetrics struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"` // in cents
}

// GetMRR calculates Monthly Recurring Revenue.
// Simplification P3: Sum of all Active Subscriptions * Plan Amount (normalized to Monthly).
func (s *AnalyticsService) GetMRR(ctx context.Context) (*MRRMetrics, error) {
	subs, err := s.subRepo.GetActiveSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	var totalMRR int64
	for _, sub := range subs {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
		if err != nil {
			// Skip or Log error? Skipping for robustness
			continue
		}
		
		// Simple Calc: Assuming 1 Price is Monthly.
		// If implementation requires multiple prices or yearly, we'd normalize here.
		if len(plan.Prices) > 0 {
			totalMRR += plan.Prices[0].Amount
		}
	}
	
	return &MRRMetrics{Currency: "USD", Amount: totalMRR}, nil
}
