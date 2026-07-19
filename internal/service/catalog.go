package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/telemetry"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type CatalogService struct {
	repo      port.PlanRepository
	telemetry *telemetry.Client // nil-safe; only set when TELEMETRY_OPTIN=true
}

// SetTelemetry injects the opt-in anonymous telemetry client after construction.
func (s *CatalogService) SetTelemetry(t *telemetry.Client) { s.telemetry = t }

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
	// HSNCode is the plan's HSN/SAC code (optional). Empty preserves the
	// tenant-SAC default at tax-resolution time.
	HSNCode string
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
		HSNCode:       input.HSNCode,
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

	s.telemetry.MilestoneFirstPlan() // opt-in anonymous milestone; no-op when disabled

	return plan, nil
}

func (s *CatalogService) ListPlans(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// GetPlan returns a single plan, or (nil, nil) if it does not exist for the tenant.
func (s *CatalogService) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.Plan, error) {
	plan, err := s.repo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.TenantID != tenantID {
		return nil, nil
	}
	return plan, nil
}

// UpdatePlanInput carries the mutable plan fields. Nil pointers are left
// unchanged, so the same call edits metadata or archives (Active=false).
type UpdatePlanInput struct {
	TenantID      uuid.UUID
	PlanID        uuid.UUID
	Name          *string
	HSNCode       *string
	IntervalUnit  *string
	IntervalCount *int
	Active        *bool
}

// UpdatePlan applies a partial update. Returns (nil, nil) when the plan does
// not exist for the tenant. Price/amount is a separate versioned entity and is
// intentionally not editable here.
func (s *CatalogService) UpdatePlan(ctx context.Context, input UpdatePlanInput) (*domain.Plan, error) {
	plan, err := s.repo.GetByID(ctx, input.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.TenantID != input.TenantID {
		return nil, nil
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		plan.Name = *input.Name
	}
	if input.HSNCode != nil {
		plan.HSNCode = *input.HSNCode
	}
	if input.IntervalUnit != nil {
		switch *input.IntervalUnit {
		case "day", "week", "month", "year":
		default:
			return nil, fmt.Errorf("interval_unit must be one of: day, week, month, year")
		}
		plan.IntervalUnit = domain.IntervalUnit(*input.IntervalUnit)
	}
	if input.IntervalCount != nil {
		if *input.IntervalCount <= 0 {
			return nil, fmt.Errorf("interval_count must be greater than 0")
		}
		plan.IntervalCount = *input.IntervalCount
	}
	if input.Active != nil {
		plan.Active = *input.Active
	}

	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}
