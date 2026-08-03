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

// TestPayInAdvanceBiller_SkipsPausedAndCanceled proves a paused or canceled
// subscription accrues NO pay-in-advance charge on a freshly-ingested usage
// event: a paused sub has billing suspended, and a canceled one is terminal
// (its final usage window closed at cancel), so a PIA capture would be a phantom
// charge — billing a customer mid-pause, or leaking an unbilled charge onto a
// dead subscription. The identical active-sub event DOES bill, isolating the
// status gate.
func TestPayInAdvanceBiller_SkipsPausedAndCanceled(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	metric := &domain.BillableMetric{ID: uuid.New(), Code: "api_calls", Name: "API calls"}
	charges := []domain.Charge{
		{ID: uuid.New(), PlanID: planID, ChargeModel: domain.ChargePerUnit, PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.0035"}}, Metric: metric},
	}
	ctx := context.Background()

	event := func(subID uuid.UUID) *domain.UsageEvent {
		return &domain.UsageEvent{ID: uuid.New(), SubscriptionID: subID, Dimension: "api_calls", Quantity: 1500}
	}

	for _, tc := range []struct {
		name    string
		status  domain.SubscriptionStatus
		wantCap int
	}{
		{"paused", domain.SubscriptionStatusPaused, 0},
		{"canceled", domain.SubscriptionStatusCanceled, 0},
		{"active", domain.SubscriptionStatusActive, 1}, // control: the gate isn't blanket-off
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: planID, Status: tc.status}
			ucRepo := &piaUnbilledRepo{}
			biller := NewPayInAdvanceBiller(&piaChargeRepo{charges: charges}, &piaPlanRepo{plan: plan}, ucRepo)

			n, err := biller.BillEvent(ctx, sub, event(sub.ID))
			if err != nil {
				t.Fatalf("BillEvent(%s): %v", tc.status, err)
			}
			if n != tc.wantCap || len(ucRepo.created) != tc.wantCap {
				t.Fatalf("%s sub: captured %d (repo %d), want %d — a %s subscription must not accrue a pay-in-advance charge",
					tc.status, n, len(ucRepo.created), tc.wantCap, tc.status)
			}
		})
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

// TestResolveChargeInput_PayInAdvanceRejectsPeriodClamps asserts a percentage
// charge that uses free_units / min_amount / max_amount (period-cumulative
// clamps) cannot also be pay_in_advance: per-event billing can't honor a
// per-period cap/floor/allowance, so it would mis-bill. Validation rejects it.
func TestResolveChargeInput_PayInAdvanceRejectsPeriodClamps(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	metric := &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "api", Name: "API", AggregationType: domain.AggregationSum}
	svc := simService(plan, metric)
	ctx := context.Background()

	for _, c := range []struct {
		name    string
		amounts domain.ChargeAmounts
	}{
		{"max_amount", domain.ChargeAmounts{Rate: "2.5", MaxAmount: 100}},
		{"min_amount", domain.ChargeAmounts{Rate: "2.5", MinAmount: 100}},
		{"free_units", domain.ChargeAmounts{Rate: "2.5", FreeUnits: 100}},
	} {
		_, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
			MetricID: metric.ID.String(), ChargeModel: "percentage", PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": c.amounts},
		})
		if err == nil {
			t.Errorf("percentage + pay_in_advance + %s should be rejected (period-cumulative clamp)", c.name)
		}
	}

	// A plain percentage (rate only) with pay_in_advance stays allowed.
	if _, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
		MetricID: metric.ID.String(), ChargeModel: "percentage", PayInAdvance: true,
		Amounts: map[string]domain.ChargeAmounts{"INR": {Rate: "2.5"}},
	}); err != nil {
		t.Fatalf("plain percentage + pay_in_advance should be allowed, got %v", err)
	}
}

// TestResolveChargeInput_PayInAdvanceRequiresAdditiveAggregation asserts
// pay_in_advance is only accepted with ADDITIVE aggregations (count, sum).
// Per-event capture bills each event independently and the captures SUM onto
// the invoice — coherent only when the period aggregate is itself a sum. For
// max/latest/unique/percentile the arrears aggregate is NOT the sum of events
// (max concurrent seats reported by heartbeat events would be billed per
// heartbeat; unique users would be billed per repeat visit), and custom
// computes an expression BillEvent never evaluates. weighted_sum was already
// rejected; this closes the rest of the non-additive set.
func TestResolveChargeInput_PayInAdvanceRequiresAdditiveAggregation(t *testing.T) {
	tenantID := uuid.New()
	planID := uuid.New()
	plan := &domain.Plan{ID: planID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}
	ctx := context.Background()

	mk := func(agg domain.AggregationType, field string) *domain.BillableMetric {
		return &domain.BillableMetric{ID: uuid.New(), TenantID: tenantID, Code: "m-" + string(agg),
			Name: string(agg), AggregationType: agg, FieldName: field}
	}

	rejected := []*domain.BillableMetric{
		mk(domain.AggregationMax, ""),
		mk(domain.AggregationLatest, ""),
		mk(domain.AggregationUnique, "user_id"),
		mk(domain.AggregationPercentile, "95"),
	}
	for _, m := range rejected {
		svc := simService(plan, m)
		_, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
			MetricID: m.ID.String(), ChargeModel: "per_unit", PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}},
		})
		if err == nil {
			t.Errorf("%s + pay_in_advance should be rejected (non-additive aggregation)", m.AggregationType)
		}
	}

	// Additive aggregations stay allowed.
	for _, agg := range []domain.AggregationType{domain.AggregationCount, domain.AggregationSum} {
		m := mk(agg, "")
		svc := simService(plan, m)
		if _, _, _, err := svc.resolveChargeInput(ctx, tenantID, 0, ChargeInput{
			MetricID: m.ID.String(), ChargeModel: "per_unit", PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}},
		}); err != nil {
			t.Errorf("%s + pay_in_advance should be allowed, got %v", agg, err)
		}
	}
}

// TestPayInAdvanceBiller_CountMetricBillsOnePerEvent asserts per-event capture
// on a COUNT metric bills exactly one unit per event, matching the arrears
// COUNT(*) semantics (quantity is ignored for count — each event counts once).
// Billing event.Quantity instead would charge a qty-5 event five units while
// the arrears path would have counted it as one.
func TestPayInAdvanceBiller_CountMetricBillsOnePerEvent(t *testing.T) {
	tenantID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: uuid.New(), Status: domain.SubscriptionStatusActive}
	plan := &domain.Plan{ID: sub.PlanID, TenantID: tenantID, Prices: []domain.Price{{Currency: "INR"}}}

	countMetric := &domain.BillableMetric{ID: uuid.New(), Code: "logins", Name: "Logins", AggregationType: domain.AggregationCount}
	charges := []domain.Charge{
		{ID: uuid.New(), PlanID: sub.PlanID, ChargeModel: domain.ChargePerUnit, PayInAdvance: true,
			Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "2"}}, Metric: countMetric},
	}

	ucRepo := &piaUnbilledRepo{}
	biller := NewPayInAdvanceBiller(&piaChargeRepo{charges: charges}, &piaPlanRepo{plan: plan}, ucRepo)

	// A count event that (incorrectly or not) carries Quantity 5 still counts
	// as ONE event: fee = 1 × ₹2 = 200p, not 5 × ₹2 = 1000p.
	n, err := biller.BillEvent(context.Background(), sub,
		&domain.UsageEvent{ID: uuid.New(), SubscriptionID: sub.ID, Dimension: "logins", Quantity: 5})
	if err != nil {
		t.Fatalf("BillEvent count: %v", err)
	}
	if n != 1 || len(ucRepo.created) != 1 {
		t.Fatalf("captured %d (repo %d), want 1", n, len(ucRepo.created))
	}
	if uc := ucRepo.created[0]; uc.Amount != 200 {
		t.Fatalf("count-metric capture = %dp, want 200p (one unit per event, matching COUNT(*) arrears)", uc.Amount)
	}
}
