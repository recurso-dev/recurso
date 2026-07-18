package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// commitmentFixture: ₹1000.00 plan, inter-state customer (18% IGST), with a
// per-period commitment in minor units.
func commitmentInvoice(t *testing.T, commitment int64, meteredQty int64) *domain.Invoice {
	t.Helper()
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"),
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	if meteredQty > 0 {
		metricID := uuid.New()
		metric := domain.BillableMetric{ID: metricID, Name: "API calls", Code: "api_calls", AggregationType: domain.AggregationSum}
		svc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
			ID: uuid.New(), PlanID: planRepo.plan.ID, MetricID: metricID,
			ChargeModel: domain.ChargePerUnit,
			Amounts:     map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.01"}}, // 1 paisa/unit
			Metric:      &metric,
		}}}
		svc.UsageRepo = &mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": meteredQty}}
		svc.RatingRepo = &mockRatingRepoForMeter{rated: map[uuid.UUID]bool{}}
	}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID,
		PlanID: planRepo.plan.ID, CommitmentAmount: commitment,
	})
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	return inv
}

func assertInvoiceReconciles(t *testing.T, inv *domain.Invoice) {
	t.Helper()
	var lineSum, lineTax int64
	for _, li := range inv.LineItems {
		lineSum += li.Amount
		lineTax += li.IGSTAmount + li.CGSTAmount + li.SGSTAmount
	}
	if lineSum != inv.Subtotal || lineTax != inv.TaxAmount || inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Fatalf("reconciliation broken: Σlines=%d subtotal=%d Σtax=%d tax=%d total=%d",
			lineSum, inv.Subtotal, lineTax, inv.TaxAmount, inv.Total)
	}
}

func hasTrueUpLine(inv *domain.Invoice) (int64, bool) {
	for _, li := range inv.LineItems {
		if strings.Contains(li.Description, "commitment true-up") {
			return li.Amount, true
		}
	}
	return 0, false
}

func TestCommitment_ShortfallFillsExactly(t *testing.T) {
	// Flat 100000 < commitment 250000 → true-up 150000.
	inv := commitmentInvoice(t, 250000, 0)
	amount, ok := hasTrueUpLine(inv)
	if !ok || amount != 150000 {
		t.Fatalf("true-up = %d/%v, want 150000/present", amount, ok)
	}
	if inv.Subtotal != 250000 {
		t.Fatalf("Subtotal = %d, want exactly the commitment 250000", inv.Subtotal)
	}
	assertInvoiceReconciles(t, inv)
}

func TestCommitment_ExactlyAtAddsNothing(t *testing.T) {
	inv := commitmentInvoice(t, 100000, 0) // flat == commitment
	if _, ok := hasTrueUpLine(inv); ok {
		t.Fatal("no true-up line may appear at exactly the commitment")
	}
	if inv.Subtotal != 100000 {
		t.Fatalf("Subtotal = %d, want 100000", inv.Subtotal)
	}
}

func TestCommitment_AboveAddsNothing(t *testing.T) {
	inv := commitmentInvoice(t, 50000, 0) // flat above commitment
	if _, ok := hasTrueUpLine(inv); ok {
		t.Fatal("no true-up line may appear above the commitment")
	}
}

func TestCommitment_UsageCountsTowardFloor(t *testing.T) {
	// Flat 100000 + usage 120000 (120000 units × 1p) = 220000 vs commitment
	// 250000 → true-up only the remaining 30000.
	inv := commitmentInvoice(t, 250000, 120000)
	amount, ok := hasTrueUpLine(inv)
	if !ok || amount != 30000 {
		t.Fatalf("true-up = %d/%v, want 30000/present (usage counts toward the floor)", amount, ok)
	}
	if inv.Subtotal != 250000 {
		t.Fatalf("Subtotal = %d, want 250000", inv.Subtotal)
	}
	assertInvoiceReconciles(t, inv)
}

func TestCommitment_UsagePastFloorNoTrueUp(t *testing.T) {
	// Flat 100000 + usage 200000 = 300000 > commitment 250000 → overage
	// bills naturally, no true-up.
	inv := commitmentInvoice(t, 250000, 200000)
	if _, ok := hasTrueUpLine(inv); ok {
		t.Fatal("no true-up when the floor is exceeded — overage bills naturally")
	}
	if inv.Subtotal != 300000 {
		t.Fatalf("Subtotal = %d, want 300000", inv.Subtotal)
	}
	assertInvoiceReconciles(t, inv)
}
