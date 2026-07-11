package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// reconcileInvoice asserts the Phase-1 money-path invariant: an invoice's line
// items sum exactly to its stored totals, with no rounding drift.
//
//	Σ line.Amount                     == invoice.Subtotal
//	Σ (line.CGST+line.SGST+line.IGST) == invoice.TaxAmount
//
// It also asserts every line carries a non-empty HSN (the IRP rejects blanks).
func reconcileInvoice(t *testing.T, inv *domain.Invoice) {
	t.Helper()
	if inv == nil {
		t.Fatal("expected an invoice, got nil")
	}
	if len(inv.LineItems) == 0 {
		t.Fatal("expected at least one line item on the invoice")
	}
	var sumAmount, sumTax int64
	for i, li := range inv.LineItems {
		if li.HSNCode == "" {
			t.Errorf("line %d has empty HSN code", i)
		}
		sumAmount += li.Amount
		sumTax += li.CGSTAmount + li.SGSTAmount + li.IGSTAmount
	}
	if sumAmount != inv.Subtotal {
		t.Errorf("Σ line.Amount = %d, want invoice.Subtotal = %d", sumAmount, inv.Subtotal)
	}
	if sumTax != inv.TaxAmount {
		t.Errorf("Σ line tax = %d, want invoice.TaxAmount = %d", sumTax, inv.TaxAmount)
	}
}

// 1.7 — recurring invoice: base plan + 2 add-ons must produce 3 reconciling lines.
func TestReconcile_RecurringInvoice_BaseAndAddons(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID, addon1ID, addon2ID := uuid.New(), uuid.New(), uuid.New()

	base := &domain.Plan{ID: basePlanID, Name: "Base", Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	addon1 := &domain.Plan{ID: addon1ID, Name: "Add-on A", Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}}
	addon2 := &domain.Plan{ID: addon2ID, Name: "Add-on B", Prices: []domain.Price{{Amount: 25000, Currency: "INR"}}}
	addons := []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addon1ID, Quantity: 1},
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addon2ID, Quantity: 2},
	}

	svc, _ := invoiceServiceWithAddons(base, []*domain.Plan{addon1, addon2}, addons)
	svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("TN"), // intra-state -> CGST+SGST
	}}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.LineItems) != 3 {
		t.Fatalf("expected 3 line items (base + 2 add-ons), got %d", len(inv.LineItems))
	}
	reconcileInvoice(t, inv)

	// Phase 1: every line uses the tenant SAC (default 998314).
	for i, li := range inv.LineItems {
		if li.HSNCode != domain.DefaultSACCode {
			t.Errorf("line %d HSN = %q, want %q (tenant SAC)", i, li.HSNCode, domain.DefaultSACCode)
		}
	}
}

// 1.7 — recurring invoice: base plan alone still emits one reconciling line and
// stays byte-identical in totals to the pre-itemization behaviour.
func TestReconcile_RecurringInvoice_BaseOnly(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID: uuid.New(), Name: "Solo", Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: uuid.New(), PlaceOfSupply: domain.StringPtr("TN"),
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(inv.LineItems))
	}
	reconcileInvoice(t, inv)
}

// 1.7 — advance invoice: single multi-period line reconciles to the totals.
func TestReconcile_AdvanceInvoice(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()

	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID: planID, Name: "Plan", IntervalUnit: domain.IntervalMonth, IntervalCount: 1,
		Prices: []domain.Price{{Amount: 10000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("TN"),
	}}
	subRepo := &mockSubRepoForInvAmt{sub: &domain.Subscription{
		ID: subID, TenantID: uuid.New(), CustomerID: customerID, PlanID: planID,
		CurrentPeriodEnd: time.Now().UTC().Add(240 * time.Hour),
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, subRepo)

	inv, err := svc.GenerateAdvanceInvoice(context.Background(), subID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(inv.LineItems))
	}
	if inv.LineItems[0].Quantity != 3 {
		t.Errorf("advance line quantity = %d, want 3 periods", inv.LineItems[0].Quantity)
	}
	reconcileInvoice(t, inv)
}

// 1.7 — subscription-initial invoice reconciles.
func TestReconcile_SubscriptionInitial(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID: planID, Name: "Starter", IntervalUnit: domain.IntervalMonth, IntervalCount: 1,
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("TN"),
	}}
	invRepo := &subMockInvoiceRepo{}
	svc := newTestSubscriptionService(&subMockSubRepo{}, invRepo, planRepo, custRepo, &subMockCouponRepo{}, &subMockGateway{})

	if _, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID: uuid.New(), CustomerID: customerID, PlanID: planID,
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reconcileInvoice(t, invRepo.created)
}

// 1.7 — proration invoice (plan upgrade) reconciles.
func TestReconcile_Proration(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	currentPlanID := uuid.New()
	newPlanID := uuid.New()
	now := time.Now().UTC()

	plans := map[uuid.UUID]*domain.Plan{
		currentPlanID: {ID: currentPlanID, Name: "Basic", IntervalUnit: domain.IntervalMonth, IntervalCount: 1, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		newPlanID:     {ID: newPlanID, Name: "Pro", IntervalUnit: domain.IntervalMonth, IntervalCount: 1, Prices: []domain.Price{{Amount: 200000, Currency: "INR"}}},
	}
	cust := &domain.Customer{ID: customerID, PlaceOfSupply: domain.StringPtr("TN")}
	subRepo := &subMockSubRepo{sub: &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID, PlanID: currentPlanID,
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
	}}
	invRepo := &subMockInvoiceRepo{}
	svc := newPreviewService(subRepo, plans, cust, invRepo)

	if _, err := svc.UpdateSubscription(context.Background(), tenantID, subRepo.sub.ID, newPlanID); err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if invRepo.created == nil {
		t.Fatal("expected a proration invoice to be created")
	}
	reconcileInvoice(t, invRepo.created)
}

// 1.7 — gift purchase invoice (single, tax-free line) reconciles.
func TestReconcile_GiftPurchase(t *testing.T) {
	planID := uuid.New()
	giftRepo := newMockGiftRepo()
	planRepo := &mockPlanRepoForGift{plan: testPlan(planID)}

	invRepo := &mockInvoiceRepoForInvAmt{}
	invSvc := newInvAmtService(invRepo,
		&mockPlanRepoForInvAmt{plan: testPlan(planID)},
		&mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: uuid.New()}},
		&mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	svc := NewGiftService(giftRepo, &mockSubRepoForGift{}, invSvc, planRepo, nil)

	if _, err := svc.PurchaseGift(context.Background(), uuid.New(), uuid.New(), planID, "r@example.com", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invRepo.created == nil {
		t.Fatal("expected a gift buyer invoice to be created")
	}
	// 1000 (unit) x 3 months = 3000, no tax.
	if invRepo.created.Subtotal != 3000 {
		t.Errorf("gift Subtotal = %d, want 3000", invRepo.created.Subtotal)
	}
	reconcileInvoice(t, invRepo.created)
}

// 1.7 — mandate debit invoice (single, tax-free line) reconciles.
func TestReconcile_MandateDebit(t *testing.T) {
	mandate := newTestMandate()
	invRepo := &mandateMockInvoiceRepo{}
	gw := &mandateMockGateway{debitResult: &port.PaymentResult{Success: true, PaymentID: "pay_x"}}
	svc := NewMandateService(&mandateMockRepo{mandate: mandate}, gw, testMandateCustomerRepo(), invRepo)

	if err := svc.ExecuteDebit(context.Background(), mandate, 500, "INR"); err != nil {
		t.Fatalf("ExecuteDebit error: %v", err)
	}
	if invRepo.created == nil {
		t.Fatal("expected a mandate debit invoice to be created")
	}
	reconcileInvoice(t, invRepo.created)
}

// Phase 2 — per-line HSN rates: a base plan at HSN 998314 (18%) plus an add-on
// plan at HSN 9972 (12%) must produce two lines taxed at DIFFERENT rates, each
// line.tax == round(line.amount × its rate), and the header tax_amount == the
// sum. This is the compliance behaviour Phase 2 delivers.
func TestReconcile_MultiRate_PerLineHSN(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID, addonID := uuid.New(), uuid.New()

	// Base plan: SaaS at 998314 -> 18%. Add-on plan: 9972 -> 12%.
	base := &domain.Plan{ID: basePlanID, Name: "SaaS", HSNCode: "998314", Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	addon := &domain.Plan{ID: addonID, Name: "Real-estate add-on", HSNCode: "9972", Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}}
	addons := []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addonID, Quantity: 1},
	}

	svc, _ := invoiceServiceWithAddons(base, []*domain.Plan{addon}, addons)
	svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("KA"), // inter-state vs org TN -> IGST
	}}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.LineItems) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(inv.LineItems))
	}

	// Base line: 100000 @ 18% = 18000. Add-on line: 50000 @ 12% = 6000.
	base0, addon1 := inv.LineItems[0], inv.LineItems[1]
	if base0.HSNCode != "998314" || base0.TaxRate != 18 {
		t.Errorf("base line HSN/rate = %q/%v, want 998314/18", base0.HSNCode, base0.TaxRate)
	}
	if addon1.HSNCode != "9972" || addon1.TaxRate != 12 {
		t.Errorf("add-on line HSN/rate = %q/%v, want 9972/12", addon1.HSNCode, addon1.TaxRate)
	}

	// Each line's tax == round(amount × rate).
	if got := base0.IGSTAmount; got != 18000 {
		t.Errorf("base line IGST = %d, want 18000 (100000×18%%)", got)
	}
	if got := addon1.IGSTAmount; got != 6000 {
		t.Errorf("add-on line IGST = %d, want 6000 (50000×12%%)", got)
	}

	// Header aggregates == the sum of the two differently-rated lines.
	if inv.Subtotal != 150000 {
		t.Errorf("Subtotal = %d, want 150000", inv.Subtotal)
	}
	if inv.TaxAmount != 24000 || inv.IGSTAmount != 24000 {
		t.Errorf("TaxAmount/IGST = %d/%d, want 24000/24000 (18000+6000)", inv.TaxAmount, inv.IGSTAmount)
	}
	if inv.Total != 174000 {
		t.Errorf("Total = %d, want 174000", inv.Total)
	}

	// Structural invariant still holds across mixed rates.
	reconcileInvoice(t, inv)
}

// Phase 2 backward-compat: a plan with an EMPTY hsn_code must tax exactly as
// today — at the tenant SAC default (998314 -> 18%). Setting the plan's HSN
// explicitly to the default SAC must produce identical numbers, proving empty
// HSN == tenant-SAC behaviour with no change.
func TestReconcile_EmptyHSN_EqualsTenantSAC(t *testing.T) {
	newSvc := func(planHSN string) *domain.Invoice {
		tenantID, customerID, subID, planID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		plan := &domain.Plan{ID: planID, Name: "Plan", HSNCode: planHSN, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
		svc, _ := invoiceServiceWithAddons(plan, nil, nil)
		svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
			ID: customerID, PlaceOfSupply: domain.StringPtr("KA"), // IGST
		}}
		inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
			ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: planID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return inv
	}

	empty := newSvc("")                       // no HSN -> tenant SAC default
	explicit := newSvc(domain.DefaultSACCode) // 998314 explicitly

	if empty.TaxAmount != 18000 || empty.IGSTAmount != 18000 {
		t.Errorf("empty-HSN tax = %d/%d, want 18000/18000 (tenant SAC 18%%)", empty.TaxAmount, empty.IGSTAmount)
	}
	if empty.TaxAmount != explicit.TaxAmount || empty.Total != explicit.Total {
		t.Errorf("empty-HSN (tax=%d,total=%d) != explicit-998314 (tax=%d,total=%d)",
			empty.TaxAmount, empty.Total, explicit.TaxAmount, explicit.Total)
	}
	// Empty HSN is still recorded on the line as the default SAC (never blank).
	if empty.LineItems[0].HSNCode != domain.DefaultSACCode {
		t.Errorf("empty-HSN line recorded HSN = %q, want %q", empty.LineItems[0].HSNCode, domain.DefaultSACCode)
	}
	reconcileInvoice(t, empty)
}

// 1.7 — e-invoice: per-item tax sums to the invoice header tax, and the
// synthetic single-line fallback is preserved for legacy (item-less) invoices.
func TestReconcile_EInvoiceItems(t *testing.T) {
	inv := &domain.Invoice{
		Subtotal: 200000, TaxAmount: 36000, IGSTAmount: 36000, Total: 236000,
		LineItems: []domain.InvoiceItem{
			{Description: "Base", HSNCode: "998314", Quantity: 1, UnitAmount: 100000, Amount: 100000, TaxRate: 18, IGSTAmount: 18000, TaxableAmount: 100000},
			{Description: "Add-on", HSNCode: "998314", Quantity: 2, UnitAmount: 50000, Amount: 100000, TaxRate: 18, IGSTAmount: 18000, TaxableAmount: 100000},
		},
	}

	items := buildEInvoiceItems(inv)
	if len(items) != 2 {
		t.Fatalf("expected 2 e-invoice items from real lines, got %d", len(items))
	}
	var sumAmount, sumTax int64
	for _, it := range items {
		sumAmount += it.TotalAmount
		sumTax += it.IGSTAmount + it.CGSTAmount + it.SGSTAmount
	}
	if sumAmount != inv.Subtotal {
		t.Errorf("Σ item amount = %d, want %d", sumAmount, inv.Subtotal)
	}
	if sumTax != inv.TaxAmount {
		t.Errorf("Σ item tax = %d, want header %d", sumTax, inv.TaxAmount)
	}

	// Legacy fallback: an invoice with no line items yields a single synthetic
	// line derived from the totals, with a non-empty HSN.
	legacy := &domain.Invoice{Subtotal: 100000, TaxAmount: 18000, IGSTAmount: 18000, Total: 118000}
	legacyItems := buildEInvoiceItems(legacy)
	if len(legacyItems) != 1 {
		t.Fatalf("legacy invoice: expected 1 synthetic item, got %d", len(legacyItems))
	}
	if legacyItems[0].HSNCode == "" {
		t.Error("legacy synthetic item must have a non-empty HSN")
	}
	if legacyItems[0].TotalAmount != legacy.Subtotal {
		t.Errorf("legacy synthetic amount = %d, want %d", legacyItems[0].TotalAmount, legacy.Subtotal)
	}
}
