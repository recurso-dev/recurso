package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// mockProgressiveRepo is an in-memory ProgressiveBillingRepository with REAL
// compare-and-swap semantics (advance only when the stored amount still equals
// old), so the flow test exercises the same idempotency the Postgres repo
// guarantees (proven separately in progressive_billing_pg_test.go).
type mockProgressiveRepo struct {
	threshold  *int64
	watermarks map[uuid.UUID]int64 // charge_id -> billed_amount (single period)
}

func (m *mockProgressiveRepo) GetThreshold(ctx context.Context, subID uuid.UUID) (*int64, error) {
	return m.threshold, nil
}
func (m *mockProgressiveRepo) ListActiveProgressiveSubscriptionIDs(ctx context.Context) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockProgressiveRepo) GetWatermark(ctx context.Context, subID, chargeID uuid.UUID, periodStart time.Time) (int64, error) {
	return m.watermarks[chargeID], nil
}
func (m *mockProgressiveRepo) AdvanceWatermarkCAS(ctx context.Context, tenantID, subID, chargeID uuid.UUID, periodStart time.Time, oldAmount, newAmount int64) (bool, error) {
	if m.watermarks[chargeID] != oldAmount {
		return false, nil // lost the CAS — already advanced
	}
	m.watermarks[chargeID] = newAmount
	return true, nil
}

// TestProgressive_InterimThenCloseBillsExactlyTotal proves the end-to-end flow:
// an interim bill plus the period-close settle together bill exactly
// rate(final) — no double-bill, no under-bill — and a retried close bills 0.
func TestProgressive_InterimThenCloseBillsExactlyTotal(t *testing.T) {
	svc, _, _, sub, _ := meteredFixture(0)

	metricID := uuid.New()
	metric := domain.BillableMetric{ID: metricID, Code: "api_calls", Name: "API calls", AggregationType: domain.AggregationSum}
	chargeID := uuid.New()
	svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:          chargeID,
		PlanID:      sub.PlanID,
		MetricID:    metricID,
		ChargeModel: domain.ChargePerUnit,
		Amounts:     map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}}, // ₹1/unit
		Metric:      &metric,
	}}}
	usage := &mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": 100}}
	svc.UsageRepo = usage
	threshold := int64(5000)
	prog := &mockProgressiveRepo{threshold: &threshold, watermarks: map[uuid.UUID]int64{}}
	svc.SetProgressiveBilling(prog, nil) // nil ledger poster: no ledger in this unit test
	ctx := context.Background()

	// Interim: cum 100 -> rate 10000p, threshold 5000 crossed -> bills 10000.
	interim, err := svc.GenerateProgressiveInvoice(ctx, sub)
	if err != nil {
		t.Fatalf("interim: %v", err)
	}
	if interim == nil || interim.Subtotal != 10000 || interim.BillingReason != domain.BillingReasonProgressiveUsage {
		t.Fatalf("interim invoice = %+v, want subtotal 10000 progressive_usage", interim)
	}
	if prog.watermarks[chargeID] != 10000 {
		t.Fatalf("watermark after interim = %d, want 10000", prog.watermarks[chargeID])
	}

	// Below-threshold interim now bills nothing (unbilled 0 < 5000).
	if inv, err := svc.GenerateProgressiveInvoice(ctx, sub); err != nil || inv != nil {
		t.Fatalf("second interim = %+v (err %v), want nil (nothing new)", inv, err)
	}

	// Close: cum grows to 250 -> the renewal invoice settles the remaining
	// 25000-10000 = 15000 on top of the ₹1000 flat fee.
	usage.qtyByMetricCode["api_calls"] = 250
	closeInv, err := svc.GenerateInvoice(ctx, sub)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closeInv.Subtotal != 115000 { // 100000 flat + 15000 progressive delta
		t.Fatalf("close subtotal = %d, want 115000 (flat 100000 + delta 15000)", closeInv.Subtotal)
	}
	if prog.watermarks[chargeID] != 25000 {
		t.Fatalf("watermark after close = %d, want 25000 == rate(250)", prog.watermarks[chargeID])
	}

	// Total progressive billed = 10000 (interim) + 15000 (close) = 25000 = rate(250). ✓

	// Retry the close: watermark already 25000, cum 250 -> delta 0 -> flat only.
	closeInv2, err := svc.GenerateInvoice(ctx, sub)
	if err != nil {
		t.Fatalf("close retry: %v", err)
	}
	if closeInv2.Subtotal != 100000 {
		t.Fatalf("retried close subtotal = %d, want 100000 (no double-bill of usage)", closeInv2.Subtotal)
	}
}

// TestProgressive_SkipsPayInAdvanceCharge proves a pay-in-advance charge is NOT
// billed by the interim progressive sweep. A PIA charge is rated per event at
// ingestion (captured as an unbilled charge); billing it again on the interim
// progressive invoice double-charges the usage. The period-close path already
// skips PIA (invoice.go meteredLines) but the interim sweep did not.
func TestProgressive_SkipsPayInAdvanceCharge(t *testing.T) {
	svc, _, _, sub, _ := meteredFixture(0)

	metricID := uuid.New()
	metric := domain.BillableMetric{ID: metricID, Code: "api_calls", Name: "API calls", AggregationType: domain.AggregationSum}
	chargeID := uuid.New()
	svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:           chargeID,
		PlanID:       sub.PlanID,
		MetricID:     metricID,
		ChargeModel:  domain.ChargePerUnit,
		PayInAdvance: true, // billed per event at ingestion; must NOT be re-billed here
		Amounts:      map[string]domain.ChargeAmounts{"INR": {UnitAmount: "1"}},
		Metric:       &metric,
	}}}
	svc.UsageRepo = &mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": 100}}
	threshold := int64(5000)
	prog := &mockProgressiveRepo{threshold: &threshold, watermarks: map[uuid.UUID]int64{}}
	svc.SetProgressiveBilling(prog, nil)

	// cum 100 x ₹1 = 10000 would cross the 5000 threshold if counted — but a
	// pay-in-advance charge is billed at ingestion, so the interim sweep must
	// bill NOTHING and advance no watermark.
	interim, err := svc.GenerateProgressiveInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("interim: %v", err)
	}
	if interim != nil {
		t.Fatalf("interim invoice subtotal = %d, want nil — a pay-in-advance charge must not be billed by the progressive sweep (already billed at ingestion)", interim.Subtotal)
	}
	if prog.watermarks[chargeID] != 0 {
		t.Errorf("watermark advanced to %d for a PIA charge, want 0 (never billed here)", prog.watermarks[chargeID])
	}
}

// TestProgressive_FilteredChargeUsesClassicPath proves a dimensional-pricing
// charge (FilterKey set) on a PROGRESSIVE subscription is billed by the classic
// filtered path — one line per value at that value's amounts — never by the
// filter-blind watermark path. The watermark path aggregates ALL events and
// rates them at the charge's BASE amounts, so routing a filtered charge through
// it silently ignores every per-value rate: us=100@₹0.02 + eu=50@₹0.03 +
// other=30@₹0.01 must bill 380p, not 180×base. Mirrors how the volume model
// (also watermark-incompatible) falls through to the classic path.
func TestProgressive_FilteredChargeUsesClassicPath(t *testing.T) {
	svc, _, ratingRepo, sub, _ := meteredFixture(0)

	metricID := uuid.New()
	metric := domain.BillableMetric{ID: metricID, Code: "api_calls", Name: "API calls", AggregationType: domain.AggregationSum}
	chargeID := uuid.New()
	svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:          chargeID,
		PlanID:      sub.PlanID,
		MetricID:    metricID,
		ChargeModel: domain.ChargePerUnit,
		Amounts:     map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.01"}}, // base rate
		FilterKey:   "region",
		Filters: []domain.ChargeFilterValue{
			{Value: "us", Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.02"}}},
			{Value: "eu", Amounts: map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.03"}}},
		},
		Metric: &metric,
	}}}
	// The blind aggregate (all regions) is 180; the per-value subsets are what
	// the filtered path bills: us=100 (200p), eu=50 (150p), other=30 (30p).
	svc.UsageRepo = &mockUsageRepoForMeter{
		qtyByMetricCode: map[string]int64{"api_calls": 180},
		filteredByValue: map[string]int64{"us": 100, "eu": 50, "__default__": 30},
	}
	threshold := int64(1) // any usage crosses it — maximal pressure on the interim path
	prog := &mockProgressiveRepo{threshold: &threshold, watermarks: map[uuid.UUID]int64{}}
	svc.SetProgressiveBilling(prog, nil)
	ctx := context.Background()

	// Interim sweep: a filtered charge is not progressively billable — nothing
	// interim, no watermark. (Blind behavior would bill 180 × ₹0.01 = 180p.)
	interim, err := svc.GenerateProgressiveInvoice(ctx, sub)
	if err != nil {
		t.Fatalf("interim: %v", err)
	}
	if interim != nil {
		t.Fatalf("interim invoice = %+v, want nil — a filtered charge must not be billed by the filter-blind watermark path", interim)
	}
	if prog.watermarks[chargeID] != 0 {
		t.Errorf("watermark advanced to %d for a filtered charge, want 0", prog.watermarks[chargeID])
	}

	// Period close: the classic filtered path bills per-value lines under one
	// rating claim, exactly as on a non-progressive subscription.
	inv, err := svc.GenerateInvoice(ctx, sub)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if inv.Subtotal != 100380 { // 100000 flat + 200 + 150 + 30
		t.Fatalf("close subtotal = %d, want 100380 (flat + per-value lines, not base-rate blind billing)", inv.Subtotal)
	}
	if len(inv.LineItems) != 4 {
		t.Fatalf("line count = %d, want 4 (base + us + eu + default)", len(inv.LineItems))
	}
	if len(ratingRepo.created) != 1 || ratingRepo.created[0].Amount != 380 {
		t.Fatalf("rating claims = %+v, want one claim amount 380", ratingRepo.created)
	}
	if prog.watermarks[chargeID] != 0 {
		t.Errorf("watermark advanced to %d at close for a filtered charge, want 0 (usage_ratings is the guard)", prog.watermarks[chargeID])
	}
}
