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
