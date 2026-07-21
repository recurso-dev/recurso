package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- fakes local to the pay-in-advance tests ---

type piaChargeRepo struct {
	port.ChargeRepository
	charges []domain.Charge
}

func (r *piaChargeRepo) ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Charge, error) {
	return r.charges, nil
}

type piaPlanRepo struct {
	port.PlanRepository
	plan *domain.Plan
}

func (r *piaPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return r.plan, nil
}

type piaUnbilledRepo struct {
	port.UnbilledChargeRepository
	created []*domain.UnbilledCharge
}

func (r *piaUnbilledRepo) Create(c *domain.UnbilledCharge) error {
	r.created = append(r.created, c)
	return nil
}

func TestPayInAdvanceBiller_BillEvent(t *testing.T) {
	tenantID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: uuid.New()}
	plan := &domain.Plan{ID: sub.PlanID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}

	perUnit := &domain.BillableMetric{ID: uuid.New(), Code: "api_calls", Name: "API calls"}
	dyn := &domain.BillableMetric{ID: uuid.New(), Code: "payments", Name: "Payments"}
	arrears := &domain.BillableMetric{ID: uuid.New(), Code: "storage", Name: "Storage"}

	charges := []domain.Charge{
		{ID: uuid.New(), PlanID: sub.PlanID, ChargeModel: domain.ChargePerUnit, PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.0035"}}, Metric: perUnit},
		{ID: uuid.New(), PlanID: sub.PlanID, ChargeModel: domain.ChargeDynamic, PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {}}, Metric: dyn},
		// arrears charge (not pay-in-advance) — never captured per event.
		{ID: uuid.New(), PlanID: sub.PlanID, ChargeModel: domain.ChargePerUnit, PayInAdvance: false,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}}, Metric: arrears},
	}

	ucRepo := &piaUnbilledRepo{}
	biller := NewPayInAdvanceBiller(&piaChargeRepo{charges: charges}, &piaPlanRepo{plan: plan}, ucRepo)
	ctx := context.Background()

	// per_unit event: 1500 × ₹0.0035 = 525p.
	n, err := biller.BillEvent(ctx, sub, &domain.UsageEvent{ID: uuid.New(), SubscriptionID: sub.ID, Dimension: "api_calls", Quantity: 1500})
	if err != nil {
		t.Fatalf("BillEvent per_unit: %v", err)
	}
	if n != 1 || len(ucRepo.created) != 1 {
		t.Fatalf("captured %d (repo %d), want 1", n, len(ucRepo.created))
	}
	if uc := ucRepo.created[0]; uc.Amount != 525 || uc.Currency != "INR" || uc.Status != domain.UnbilledChargeStatusPending || uc.SubscriptionID != sub.ID {
		t.Fatalf("unbilled charge = %+v, want 525 INR pending on sub", uc)
	}

	// dynamic event: bills the event's dynamic_amount (4200p).
	n, err = biller.BillEvent(ctx, sub, &domain.UsageEvent{ID: uuid.New(), SubscriptionID: sub.ID, Dimension: "payments", Quantity: 1, DynamicAmount: 4200})
	if err != nil {
		t.Fatalf("BillEvent dynamic: %v", err)
	}
	if n != 1 || ucRepo.created[1].Amount != 4200 {
		t.Fatalf("dynamic capture = %d / %d, want 1 / 4200", n, ucRepo.created[1].Amount)
	}

	// arrears dimension: nothing captured per event.
	n, err = biller.BillEvent(ctx, sub, &domain.UsageEvent{ID: uuid.New(), SubscriptionID: sub.ID, Dimension: "storage", Quantity: 10})
	if err != nil {
		t.Fatalf("BillEvent arrears: %v", err)
	}
	if n != 0 || len(ucRepo.created) != 2 {
		t.Fatalf("arrears event captured %d (repo %d), want 0 (repo 2)", n, len(ucRepo.created))
	}
}

// TestSetPlanCharges_PayInAdvanceRejectsCumulativeModels asserts the validation
// restricting pay_in_advance to non-cumulative models.
func TestResolveChargeInput_PayInAdvanceModelRestriction(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	metric := &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "api", Name: "API", AggregationType: domain.AggregationSum}
	svc := simService(plan, metric) // reuses simulator-test fakes (metrics+plans)
	ctx := context.Background()

	tiers := []domain.ChargeTier{{UpTo: nil, UnitAmount: "1"}}

	// graduated + pay_in_advance -> rejected.
	_, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
		MetricID: metric.ID.String(), ChargeModel: "graduated", PayInAdvance: true,
		Amounts: map[string]domain.ChargeAmounts{"INR": {Tiers: tiers}},
	})
	if err == nil {
		t.Fatal("graduated + pay_in_advance should be rejected")
	}

	// per_unit + pay_in_advance -> allowed.
	if _, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
		MetricID: metric.ID.String(), ChargeModel: "per_unit", PayInAdvance: true,
		Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.5"}},
	}); err != nil {
		t.Fatalf("per_unit + pay_in_advance should be allowed, got %v", err)
	}
}
