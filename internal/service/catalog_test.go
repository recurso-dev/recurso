package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mock PlanRepository ---

type mockPlanRepo struct {
	port.PlanRepository
	created *domain.Plan
}

func (m *mockPlanRepo) Create(ctx context.Context, plan *domain.Plan) error {
	m.created = plan
	return nil
}

// --- Tests ---

func TestCreatePlan_Success(t *testing.T) {
	repo := &mockPlanRepo{}
	svc := NewCatalogService(repo)

	plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:      uuid.New(),
		Name:          "Pro Monthly",
		Code:          "pro_monthly",
		IntervalUnit:  "month",
		IntervalCount: 1,
		Amount:        9900,
		Currency:      "USD",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ID == uuid.Nil {
		t.Error("plan ID should be generated")
	}
	if plan.Name != "Pro Monthly" {
		t.Errorf("name = %q, want 'Pro Monthly'", plan.Name)
	}
	if len(plan.Prices) != 1 || plan.Prices[0].Amount != 9900 {
		t.Error("expected one price with amount 9900")
	}
	if repo.created == nil {
		t.Error("expected repo.Create to be called")
	}
}

func TestCreatePlan_FreePlan(t *testing.T) {
	repo := &mockPlanRepo{}
	svc := NewCatalogService(repo)

	plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
		TenantID:      uuid.New(),
		Name:          "Free",
		Code:          "free",
		IntervalUnit:  "month",
		IntervalCount: 1,
		Amount:        0,
		Currency:      "USD",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Prices[0].Amount != 0 {
		t.Errorf("amount = %d, want 0 for free plan", plan.Prices[0].Amount)
	}
}

func TestCreatePlan_Validation(t *testing.T) {
	valid := CreatePlanInput{
		TenantID:      uuid.New(),
		Name:          "Pro",
		Code:          "pro",
		IntervalUnit:  "month",
		IntervalCount: 1,
		Amount:        1000,
		Currency:      "USD",
	}

	tests := []struct {
		name    string
		modify  func(*CreatePlanInput)
		wantErr string
	}{
		{
			name:    "negative amount",
			modify:  func(i *CreatePlanInput) { i.Amount = -100 },
			wantErr: "amount cannot be negative",
		},
		{
			name:    "empty name",
			modify:  func(i *CreatePlanInput) { i.Name = "" },
			wantErr: "name is required",
		},
		{
			name:    "empty code",
			modify:  func(i *CreatePlanInput) { i.Code = "" },
			wantErr: "code is required",
		},
		{
			name:    "currency too short",
			modify:  func(i *CreatePlanInput) { i.Currency = "US" },
			wantErr: "currency must be a 3-letter code",
		},
		{
			name:    "currency too long",
			modify:  func(i *CreatePlanInput) { i.Currency = "USDX" },
			wantErr: "currency must be a 3-letter code",
		},
		{
			name:    "empty currency",
			modify:  func(i *CreatePlanInput) { i.Currency = "" },
			wantErr: "currency must be a 3-letter code",
		},
		{
			name:    "invalid interval unit",
			modify:  func(i *CreatePlanInput) { i.IntervalUnit = "hourly" },
			wantErr: "interval_unit must be one of: day, week, month, year",
		},
		{
			name:    "empty interval unit",
			modify:  func(i *CreatePlanInput) { i.IntervalUnit = "" },
			wantErr: "interval_unit must be one of: day, week, month, year",
		},
		{
			name:    "zero interval count",
			modify:  func(i *CreatePlanInput) { i.IntervalCount = 0 },
			wantErr: "interval_count must be greater than 0",
		},
		{
			name:    "negative interval count",
			modify:  func(i *CreatePlanInput) { i.IntervalCount = -1 },
			wantErr: "interval_count must be greater than 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.modify(&input)

			repo := &mockPlanRepo{}
			svc := NewCatalogService(repo)

			_, err := svc.CreatePlan(context.Background(), input)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tc.wantErr)
			}
			if repo.created != nil {
				t.Error("repo.Create should not be called on validation failure")
			}
		})
	}
}

func TestCreatePlan_AllIntervalUnits(t *testing.T) {
	units := []string{"day", "week", "month", "year"}

	for _, unit := range units {
		t.Run(unit, func(t *testing.T) {
			repo := &mockPlanRepo{}
			svc := NewCatalogService(repo)

			plan, err := svc.CreatePlan(context.Background(), CreatePlanInput{
				TenantID:      uuid.New(),
				Name:          "Plan " + unit,
				Code:          "plan_" + unit,
				IntervalUnit:  unit,
				IntervalCount: 1,
				Amount:        500,
				Currency:      "INR",
			})
			if err != nil {
				t.Fatalf("unexpected error for unit %q: %v", unit, err)
			}
			if string(plan.IntervalUnit) != unit {
				t.Errorf("interval_unit = %q, want %q", plan.IntervalUnit, unit)
			}
		})
	}
}
