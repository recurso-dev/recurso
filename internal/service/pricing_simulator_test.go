package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Minimal fakes: SimulateCharges (no subscription) uses only plans.GetByID
// and metrics.GetByID; the charge/subscription/usage repos are never called. ---

type simPlanRepo struct {
	port.PlanRepository
	plan *domain.Plan
}

func (r *simPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return r.plan, nil
}

type simMetricRepo struct {
	port.BillableMetricRepository
	byID map[uuid.UUID]*domain.BillableMetric
}

func (r *simMetricRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BillableMetric, error) {
	return r.byID[id], nil
}

func simService(plan *domain.Plan, metrics ...*domain.BillableMetric) *MeteringService {
	byID := map[uuid.UUID]*domain.BillableMetric{}
	for _, m := range metrics {
		byID[m.ID] = m
	}
	return NewMeteringService(&simMetricRepo{byID: byID}, nil, &simPlanRepo{plan: plan}, nil, nil)
}

func TestSimulateCharges_RatesAndPreviewsGL(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR", Amount: 100000}}}

	m1 := &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "api_calls", Name: "API calls", AggregationType: domain.AggregationSum}
	m2 := &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "pay_volume", Name: "Payment volume", AggregationType: domain.AggregationSum}
	svc := simService(plan, m1, m2)

	req := SimulateRequest{
		Charges: []ChargeInput{
			{MetricID: m1.ID.String(), ChargeModel: "per_unit", Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.0035"}}},
			{MetricID: m2.ID.String(), ChargeModel: "percentage", Amounts: map[string]domain.ChargeAmounts{"INR": {Rate: "2.5"}}},
		},
		Usage: []SimulateUsage{
			{MetricID: m1.ID.String(), Quantity: 1500},   // 1500 × ₹0.0035 = 525p
			{MetricID: m2.ID.String(), Quantity: 100000}, // 2.5% × 100000 = 2500p
		},
	}

	sim, err := svc.SimulateCharges(context.Background(), tenantID, planID, req)
	if err != nil {
		t.Fatalf("SimulateCharges: %v", err)
	}
	if sim.Currency != "INR" {
		t.Fatalf("currency = %q, want INR (from plan price)", sim.Currency)
	}
	if len(sim.Charges) != 2 {
		t.Fatalf("charges = %d, want 2", len(sim.Charges))
	}
	if sim.Charges[0].Amount != 525 || sim.Charges[1].Amount != 2500 {
		t.Fatalf("amounts = %d, %d; want 525, 2500", sim.Charges[0].Amount, sim.Charges[1].Amount)
	}
	if sim.Subtotal != 3025 {
		t.Fatalf("subtotal = %d, want 3025", sim.Subtotal)
	}

	// GL preview: DR AR 3025 / CR Revenue 3025, balanced.
	var dr, cr int64
	for _, l := range sim.GLPreview {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr || dr != 3025 {
		t.Fatalf("GL preview DR=%d CR=%d, want 3025==3025", dr, cr)
	}
	if !sim.Balanced {
		t.Fatal("simulation should be balanced")
	}
}

func TestSimulateCharges_RejectsBadConfigAndUnknownPlan(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	m := &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "api", Name: "API", AggregationType: domain.AggregationCount}

	// Unknown/other-tenant plan -> not found.
	otherPlan := &domain.Plan{ID: planID, TenantID: uuid.New(), Prices: []domain.Price{{Currency: "INR"}}}
	svc := simService(otherPlan, m)
	if _, err := svc.SimulateCharges(context.Background(), tenantID, planID, SimulateRequest{}); err != ErrMeteringPlanNotFound {
		t.Fatalf("want ErrMeteringPlanNotFound, got %v", err)
	}

	// Bad pricing config (percentage with no rate) -> validation error.
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	svc2 := simService(plan, m)
	_, err := svc2.SimulateCharges(context.Background(), tenantID, planID, SimulateRequest{
		Charges: []ChargeInput{{MetricID: m.ID.String(), ChargeModel: "percentage", Amounts: map[string]domain.ChargeAmounts{"INR": {}}}},
	})
	var valErr MeteringValidationError
	if err == nil || !errorsAs(err, &valErr) {
		t.Fatalf("want MeteringValidationError for missing rate, got %v", err)
	}
}

// errorsAs is a tiny local helper to avoid importing errors just for one As.
func errorsAs(err error, target *MeteringValidationError) bool {
	if v, ok := err.(MeteringValidationError); ok {
		*target = v
		return true
	}
	return false
}
