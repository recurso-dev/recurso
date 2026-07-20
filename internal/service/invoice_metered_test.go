package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Fakes for metered-line tests (usage-based billing v1) ---

type mockChargeRepoForMeter struct {
	port.ChargeRepository
	charges []domain.Charge
}

func (m *mockChargeRepoForMeter) ListByPlan(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Charge, error) {
	return m.charges, nil
}

type mockUsageRepoForMeter struct {
	port.UsageRepository
	qtyByMetricCode map[string]int64
	dynByDimension  map[string]int64
	gotStart        time.Time
	gotEnd          time.Time
}

func (m *mockUsageRepoForMeter) AggregateForMetric(ctx context.Context, subscriptionID uuid.UUID, metric domain.BillableMetric, start, end time.Time) (int64, error) {
	m.gotStart, m.gotEnd = start, end
	return m.qtyByMetricCode[metric.Code], nil
}

func (m *mockUsageRepoForMeter) SumDynamicAmount(ctx context.Context, subscriptionID uuid.UUID, dimension string, start, end time.Time) (int64, error) {
	m.gotStart, m.gotEnd = start, end
	return m.dynByDimension[dimension], nil
}

type mockRatingRepoForMeter struct {
	port.UsageRatingRepository
	rated   map[uuid.UUID]bool // chargeID -> already rated
	created []*domain.UsageRating
}

func (m *mockRatingRepoForMeter) Exists(ctx context.Context, subscriptionID, chargeID uuid.UUID, periodStart time.Time) (bool, error) {
	return m.rated[chargeID], nil
}

func (m *mockRatingRepoForMeter) Create(ctx context.Context, r *domain.UsageRating) (bool, error) {
	m.created = append(m.created, r)
	return true, nil
}

// meteredFixture builds an InvoiceService wired with one per_unit charge:
// metric "api_calls" (sum) at ₹0.0035/call in INR on a ₹1000.00 plan, for an
// inter-state (IGST) customer.
func meteredFixture(qty int64) (*InvoiceService, *mockInvoiceRepoForInvAmt, *mockRatingRepoForMeter, *domain.Subscription, uuid.UUID) {
	metricID, chargeID := uuid.New(), uuid.New()
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"), // inter-state vs org TN -> IGST
	}}
	invRepo := &mockInvoiceRepoForInvAmt{}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	metric := domain.BillableMetric{
		ID: metricID, Name: "API calls", Code: "api_calls",
		AggregationType: domain.AggregationSum,
	}
	svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:          chargeID,
		PlanID:      planRepo.plan.ID,
		MetricID:    metricID,
		ChargeModel: domain.ChargePerUnit,
		Amounts:     map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.0035"}},
		Metric:      &metric,
	}}}
	svc.UsageRepo = &mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": qty}}
	ratingRepo := &mockRatingRepoForMeter{rated: map[uuid.UUID]bool{}}
	svc.RatingRepo = ratingRepo

	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		CustomerID:         custRepo.customer.ID,
		PlanID:             planRepo.plan.ID,
		CurrentPeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	return svc, invRepo, ratingRepo, sub, chargeID
}

// TestGenerateInvoice_DynamicCharge asserts a dynamic charge bills the summed
// per-event dynamic_amount (via SumDynamicAmount), not the metric aggregation.
func TestGenerateInvoice_DynamicCharge(t *testing.T) {
	svc, invRepo, ratingRepo, sub, _ := meteredFixture(0)

	metricID := uuid.New()
	metric := domain.BillableMetric{
		ID: metricID, Name: "Payments", Code: "payments",
		AggregationType: domain.AggregationSum,
	}
	dynChargeID := uuid.New()
	svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:          dynChargeID,
		PlanID:      sub.PlanID,
		MetricID:    metricID,
		ChargeModel: domain.ChargeDynamic,
		Amounts:     map[string]domain.ChargeAmounts{"INR": {}},
		Metric:      &metric,
	}}}
	// Aggregation would return 0 (empty qtyByMetricCode); the dynamic sum is 4200p.
	svc.UsageRepo = &mockUsageRepoForMeter{dynByDimension: map[string]int64{"payments": 4200}}

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 100000p base + 4200p dynamic usage.
	if inv.Subtotal != 104200 {
		t.Fatalf("Subtotal = %d, want 104200", inv.Subtotal)
	}
	if len(inv.LineItems) != 2 {
		t.Fatalf("line count = %d, want 2 (base + dynamic)", len(inv.LineItems))
	}
	metered := inv.LineItems[1]
	if metered.Amount != 4200 {
		t.Fatalf("dynamic line = %d, want 4200 (the dynamic_amount sum)", metered.Amount)
	}
	// The rating claim records the dynamic sum as the billed quantity and amount.
	if len(ratingRepo.created) != 1 || ratingRepo.created[0].Quantity != 4200 || ratingRepo.created[0].Amount != 4200 {
		t.Fatalf("claim = %+v, want qty 4200 amount 4200", ratingRepo.created)
	}
	if invRepo.created == nil {
		t.Fatal("invoice was not persisted")
	}
}

func TestGenerateInvoice_MeteredLineAddedWithInvariants(t *testing.T) {
	svc, invRepo, ratingRepo, sub, chargeID := meteredFixture(1500)

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1500 × ₹0.0035 = ₹5.25 = 525p usage on top of the 100000p base.
	if inv.Subtotal != 100525 {
		t.Fatalf("Subtotal = %d, want 100525", inv.Subtotal)
	}
	// Per-line 18% IGST, each line rounded by the GST engine:
	// 100000×0.18 = 18000 and round(525×0.18) = round(94.5) = 95.
	if inv.TaxAmount != 18095 || inv.IGSTAmount != 18095 {
		t.Fatalf("Tax/IGST = %d/%d, want 18095/18095", inv.TaxAmount, inv.IGSTAmount)
	}
	if inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Fatalf("Total (%d) != Subtotal (%d) + Tax (%d)", inv.Total, inv.Subtotal, inv.TaxAmount)
	}

	// Exact-reconciliation invariants over the itemized lines.
	if len(inv.LineItems) != 2 {
		t.Fatalf("line count = %d, want 2 (base + metered)", len(inv.LineItems))
	}
	var lineSum, lineTax int64
	for _, li := range inv.LineItems {
		lineSum += li.Amount
		lineTax += li.IGSTAmount + li.CGSTAmount + li.SGSTAmount
	}
	if lineSum != inv.Subtotal {
		t.Fatalf("Σ line.Amount = %d, want Subtotal %d", lineSum, inv.Subtotal)
	}
	if lineTax != inv.TaxAmount {
		t.Fatalf("Σ line tax = %d, want TaxAmount %d", lineTax, inv.TaxAmount)
	}

	metered := inv.LineItems[1]
	if metered.Amount != 525 || metered.Quantity != 1 || metered.UnitAmount != 525 {
		t.Fatalf("metered line = qty %d × %d = %d, want 1 × 525 = 525", metered.Quantity, metered.UnitAmount, metered.Amount)
	}
	if !strings.Contains(metered.Description, "API calls") || !strings.Contains(metered.Description, "1500") {
		t.Fatalf("metered description %q should carry the metric name and usage count", metered.Description)
	}

	// The rating claim is persisted against the committed invoice.
	if len(ratingRepo.created) != 1 {
		t.Fatalf("rating claims = %d, want 1", len(ratingRepo.created))
	}
	claim := ratingRepo.created[0]
	if claim.ChargeID != chargeID || claim.InvoiceID != inv.ID || claim.Quantity != 1500 || claim.Amount != 525 {
		t.Fatalf("claim = %+v, want charge %s on invoice %s qty 1500 amount 525", claim, chargeID, inv.ID)
	}
	if !claim.PeriodStart.Equal(sub.CurrentPeriodStart) || !claim.PeriodEnd.Equal(sub.CurrentPeriodEnd) {
		t.Fatalf("claim window = %v–%v, want the subscription's current period", claim.PeriodStart, claim.PeriodEnd)
	}
	if invRepo.created == nil {
		t.Fatal("invoice was not persisted")
	}
}

func TestGenerateInvoice_MeteredWindowIsElapsedPeriod(t *testing.T) {
	svc, _, _, sub, _ := meteredFixture(10)
	usageRepo := svc.UsageRepo.(*mockUsageRepoForMeter)

	if _, err := svc.GenerateInvoice(context.Background(), sub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !usageRepo.gotStart.Equal(sub.CurrentPeriodStart) || !usageRepo.gotEnd.Equal(sub.CurrentPeriodEnd) {
		t.Fatalf("aggregated window %v–%v, want %v–%v (arrears over the elapsed period)",
			usageRepo.gotStart, usageRepo.gotEnd, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	}
}

func TestGenerateInvoice_AlreadyRatedWindowSkipped(t *testing.T) {
	svc, _, ratingRepo, sub, chargeID := meteredFixture(1500)
	ratingRepo.rated[chargeID] = true // retried generation: window already billed

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Subtotal != 100000 {
		t.Fatalf("Subtotal = %d, want 100000 (no metered line on retry)", inv.Subtotal)
	}
	if len(inv.LineItems) != 1 {
		t.Fatalf("line count = %d, want 1", len(inv.LineItems))
	}
	if len(ratingRepo.created) != 0 {
		t.Fatalf("rating claims = %d, want 0", len(ratingRepo.created))
	}
}

func TestGenerateInvoice_ZeroUsageEmitsNoLineAndNoClaim(t *testing.T) {
	svc, _, ratingRepo, sub, _ := meteredFixture(0)

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Subtotal != 100000 || len(inv.LineItems) != 1 {
		t.Fatalf("Subtotal/lines = %d/%d, want 100000/1 (zero usage bills nothing)", inv.Subtotal, len(inv.LineItems))
	}
	// No claim either: late-arriving events for the window can still bill later.
	if len(ratingRepo.created) != 0 {
		t.Fatalf("rating claims = %d, want 0", len(ratingRepo.created))
	}
}

func TestGenerateInvoice_ChargeWithoutInvoiceCurrencySkipped(t *testing.T) {
	svc, _, ratingRepo, sub, _ := meteredFixture(1500)
	// Reprice the charge in USD only — the INR invoice must skip it.
	chargeRepo := svc.ChargeRepo.(*mockChargeRepoForMeter)
	chargeRepo.charges[0].Amounts = map[string]domain.ChargeAmounts{"USD": {UnitAmount: "0.0035"}}

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Subtotal != 100000 || len(inv.LineItems) != 1 {
		t.Fatalf("Subtotal/lines = %d/%d, want 100000/1 (currency-less charge skipped)", inv.Subtotal, len(inv.LineItems))
	}
	if len(ratingRepo.created) != 0 {
		t.Fatalf("rating claims = %d, want 0", len(ratingRepo.created))
	}
}

func TestGenerateFinalUsageInvoice_BillsPartialWindow(t *testing.T) {
	svc, invRepo, ratingRepo, sub, _ := meteredFixture(1500)
	endedAt := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC) // mid-period cancel

	inv, err := svc.GenerateFinalUsageInvoice(context.Background(), sub, endedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected a final usage invoice")
	}
	// Usage only — no flat fee (it was billed in advance at period start).
	if inv.Subtotal != 525 || len(inv.LineItems) != 1 {
		t.Fatalf("Subtotal/lines = %d/%d, want 525/1 (usage only)", inv.Subtotal, len(inv.LineItems))
	}
	if inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Fatalf("Total (%d) != Subtotal (%d) + Tax (%d)", inv.Total, inv.Subtotal, inv.TaxAmount)
	}
	usageRepo := svc.UsageRepo.(*mockUsageRepoForMeter)
	if !usageRepo.gotStart.Equal(sub.CurrentPeriodStart) || !usageRepo.gotEnd.Equal(endedAt) {
		t.Fatalf("aggregated window %v–%v, want %v–%v (partial elapsed window)",
			usageRepo.gotStart, usageRepo.gotEnd, sub.CurrentPeriodStart, endedAt)
	}
	if len(ratingRepo.created) != 1 || !ratingRepo.created[0].PeriodEnd.Equal(endedAt) {
		t.Fatalf("expected one rating claim ending at the cancel time, got %+v", ratingRepo.created)
	}
	if invRepo.created == nil {
		t.Fatal("final invoice was not persisted")
	}
}

func TestGenerateFinalUsageInvoice_NoUsageNoInvoice(t *testing.T) {
	svc, invRepo, ratingRepo, sub, _ := meteredFixture(0)

	inv, err := svc.GenerateFinalUsageInvoice(context.Background(), sub, time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv != nil {
		t.Fatalf("expected no invoice for zero usage, got %+v", inv)
	}
	if invRepo.created != nil || len(ratingRepo.created) != 0 {
		t.Fatal("nothing should be persisted for zero usage")
	}
}

func TestGenerateInvoice_NoRatingClaimWhenInvoicePersistFails(t *testing.T) {
	svc, invRepo, ratingRepo, sub, _ := meteredFixture(1500)
	invRepo.createErr = errors.New("insert failed")

	if _, err := svc.GenerateInvoice(context.Background(), sub); err == nil {
		t.Fatal("expected error when invoice persistence fails")
	}
	// The window must stay unclaimed so the retry can bill it.
	if len(ratingRepo.created) != 0 {
		t.Fatalf("rating claims = %d after failed persist, want 0", len(ratingRepo.created))
	}
}
