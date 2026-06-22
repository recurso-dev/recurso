package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type CatalogService struct {
	repo port.PlanRepository
}

func NewCatalogService(repo port.PlanRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

type CreatePlanInput struct {
	TenantID      uuid.UUID
	Name          string
	Code          string
	IntervalUnit  string
	IntervalCount int
	Amount        int64
	Currency      string
}

func (s *CatalogService) CreatePlan(ctx context.Context, input CreatePlanInput) (*domain.Plan, error) {
	// 1. Validation
	if input.Amount < 0 {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Code == "" {
		return nil, fmt.Errorf("code is required")
	}
	if len(input.Currency) != 3 {
		return nil, fmt.Errorf("currency must be a 3-letter code")
	}
	switch input.IntervalUnit {
	case "day", "week", "month", "year":
	default:
		return nil, fmt.Errorf("interval_unit must be one of: day, week, month, year")
	}
	if input.IntervalCount <= 0 {
		return nil, fmt.Errorf("interval_count must be greater than 0")
	}

	now := time.Now().UTC()
	planID := uuid.New()

	plan := &domain.Plan{
		ID:            planID,
		TenantID:      input.TenantID,
		Name:          input.Name,
		Code:          input.Code,
		IntervalUnit:  domain.IntervalUnit(input.IntervalUnit),
		IntervalCount: input.IntervalCount,
		Active:        true,
		CreatedAt:     now,
		Prices: []domain.Price{
			{
				ID:        uuid.New(),
				PlanID:    planID,
				Currency:  input.Currency,
				Amount:    input.Amount,
				Type:      "recurring",
				CreatedAt: now,
			},
		},
	}

	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *CatalogService) ListPlans(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	return s.repo.List(ctx, tenantID, filter)
}
