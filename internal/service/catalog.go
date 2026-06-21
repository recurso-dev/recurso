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
	// TODO: Add thorough validation

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
