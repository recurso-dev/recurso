package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// reconcileLines asserts the Phase-3 money-path invariants on a set of lines
// against an invoice's discount and stored totals:
//
//	Σ line.Amount        == subtotal (gross)
//	Σ line.TaxableAmount == subtotal − discount
//	Σ line tax           == taxAmount (== IGST+CGST+SGST header)
func reconcileLines(t *testing.T, lines []domain.InvoiceItem, subtotal, discount, taxAmount int64) {
	t.Helper()
	var sumAmount, sumTaxable, sumTax int64
	for i, li := range lines {
		if li.HSNCode == "" {
			t.Errorf("line %d has empty HSN", i)
		}
		sumAmount += li.Amount
		sumTaxable += li.TaxableAmount
		sumTax += li.CGSTAmount + li.SGSTAmount + li.IGSTAmount
	}
	if sumAmount != subtotal {
		t.Errorf("Σ line.Amount = %d, want subtotal %d", sumAmount, subtotal)
	}
	if sumTaxable != subtotal-discount {
		t.Errorf("Σ line.TaxableAmount = %d, want subtotal-discount %d", sumTaxable, subtotal-discount)
	}
	if sumTax != taxAmount {
		t.Errorf("Σ line tax = %d, want tax_amount %d", sumTax, taxAmount)
	}
}

// Task A — the multi-line discount rule: pro-rata by gross amount with
// largest-remainder rounding so the shares sum to the discount exactly, each
// line's tax recomputed on its post-discount taxable base.
//
// 2-line intra-state invoice, gross 150000 (100000 + 50000), discount 10000.
//
//	shares (largest-remainder): line0 6667, line1 3333 (Σ == 10000)
//	taxable: line0 93333, line1 46667 (Σ == 140000 == 150000-10000)
//	tax @18%: line0 trunc(93333*.18)=16799 -> CGST 8399 / SGST 8400
//	          line1 trunc(46667*.18)=8400  -> CGST 4200 / SGST 4200
//	header: CGST 12599, SGST 12600, total 25199
func TestDistributeDiscount_MultiLine_IntraState(t *testing.T) {
	lines := []domain.InvoiceItem{
		{Description: "Base", HSNCode: "998314", Amount: 100000, TaxRate: 18, CGSTAmount: 9000, SGSTAmount: 9000},
		{Description: "Add-on", HSNCode: "998314", Amount: 50000, TaxRate: 18, CGSTAmount: 4500, SGSTAmount: 4500},
	}

	igst, cgst, sgst, total := distributeDiscount(lines, 10000)

	if lines[0].TaxableAmount != 93333 || lines[1].TaxableAmount != 46667 {
		t.Fatalf("taxable = %d/%d, want 93333/46667", lines[0].TaxableAmount, lines[1].TaxableAmount)
	}
	// Largest-remainder: line0's remainder (.67) beats line1's (.33) for the
	// single leftover paisa, so line0's share is 6667 not 6666.
	if got := lines[0].Amount - lines[0].TaxableAmount; got != 6667 {
		t.Errorf("line0 discount share = %d, want 6667", got)
	}
	if lines[0].CGSTAmount != 8399 || lines[0].SGSTAmount != 8400 {
		t.Errorf("line0 CGST/SGST = %d/%d, want 8399/8400", lines[0].CGSTAmount, lines[0].SGSTAmount)
	}
	if lines[1].CGSTAmount != 4200 || lines[1].SGSTAmount != 4200 {
		t.Errorf("line1 CGST/SGST = %d/%d, want 4200/4200", lines[1].CGSTAmount, lines[1].SGSTAmount)
	}
	if igst != 0 || cgst != 12599 || sgst != 12600 || total != 25199 {
		t.Errorf("header igst/cgst/sgst/total = %d/%d/%d/%d, want 0/12599/12600/25199", igst, cgst, sgst, total)
	}

	// Per line: tax == trunc(taxable × rate) (the engine rule).
	for i, li := range lines {
		want := int64(float64(li.TaxableAmount) * (li.TaxRate / 100.0))
		if got := li.CGSTAmount + li.SGSTAmount + li.IGSTAmount; got != want {
			t.Errorf("line %d tax = %d, want trunc(taxable×rate) = %d", i, got, want)
		}
	}

	reconcileLines(t, lines, 150000, 10000, total)
}

// Task A — inter-state (IGST) multi-line distribution reconciles the same way.
func TestDistributeDiscount_MultiLine_InterState(t *testing.T) {
	lines := []domain.InvoiceItem{
		{HSNCode: "998314", Amount: 100000, TaxRate: 18, IGSTAmount: 18000},
		{HSNCode: "9972", Amount: 50000, TaxRate: 12, IGSTAmount: 6000},
	}
	igst, cgst, sgst, total := distributeDiscount(lines, 30000)

	// shares: line0 30000*100000/150000=20000, line1 10000 (exact, no remainder)
	if lines[0].TaxableAmount != 80000 || lines[1].TaxableAmount != 40000 {
		t.Fatalf("taxable = %d/%d, want 80000/40000", lines[0].TaxableAmount, lines[1].TaxableAmount)
	}
	// line0 @18% on 80000 = 14400; line1 @12% on 40000 = 4800.
	if lines[0].IGSTAmount != 14400 || lines[1].IGSTAmount != 4800 {
		t.Errorf("line IGST = %d/%d, want 14400/4800", lines[0].IGSTAmount, lines[1].IGSTAmount)
	}
	if cgst != 0 || sgst != 0 || igst != 19200 || total != 19200 {
		t.Errorf("header = igst %d cgst %d sgst %d total %d, want 19200/0/0/19200", igst, cgst, sgst, total)
	}
	reconcileLines(t, lines, 150000, 30000, total)
}

// Task A — the single-line discounted invoice (the only discounted path in
// production today) records the post-discount taxable base while keeping the
// gross Amount and the engine-computed header tax verbatim — no total shift.
func TestCreateSubscription_DiscountedLine_TaxableBase(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()

	planRepo := &subMockPlanRepo{plan: &domain.Plan{
		ID: planID, Name: "Pro", IntervalUnit: domain.IntervalMonth, IntervalCount: 1,
		Prices: []domain.Price{{Amount: 10000, Currency: "INR"}},
	}}
	custRepo := &subMockCustomerRepo{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("TN"), // intra-state -> CGST+SGST
	}}
	invRepo := &subMockInvoiceRepo{}
	couponRepo := &subMockCouponRepo{coupon: &domain.Coupon{
		ID: uuid.New(), Code: "HALF", DiscountType: domain.DiscountTypePercent, DiscountValue: 50,
	}}
	svc := newTestSubscriptionService(&subMockSubRepo{}, invRepo, planRepo, custRepo, couponRepo, &subMockGateway{})

	if _, err := svc.CreateSubscription(context.Background(), CreateSubscriptionInput{
		TenantID: uuid.New(), CustomerID: customerID, PlanID: planID,
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), CouponCode: "HALF",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv := invRepo.created
	if inv == nil {
		t.Fatal("expected invoice")
	}

	// Totals unchanged vs pre-Phase-3 behaviour: gross subtotal 10000, tax on the
	// post-discount 5000 base = 900, total 5900.
	if inv.Subtotal != 10000 || inv.TaxAmount != 900 || inv.Total != 5900 {
		t.Errorf("subtotal/tax/total = %d/%d/%d, want 10000/900/5900", inv.Subtotal, inv.TaxAmount, inv.Total)
	}
	if len(inv.LineItems) != 1 {
		t.Fatalf("expected 1 line, got %d", len(inv.LineItems))
	}
	li := inv.LineItems[0]
	if li.Amount != 10000 {
		t.Errorf("line.Amount = %d, want 10000 (gross)", li.Amount)
	}
	if li.TaxableAmount != 5000 {
		t.Errorf("line.TaxableAmount = %d, want 5000 (post-discount base)", li.TaxableAmount)
	}
	if got := li.CGSTAmount + li.SGSTAmount + li.IGSTAmount; got != 900 {
		t.Errorf("line tax = %d, want 900", got)
	}
	// line.tax == trunc(taxable × rate): 5000 × 18% = 900.
	if want := int64(float64(li.TaxableAmount) * (li.TaxRate / 100.0)); want != 900 {
		t.Errorf("trunc(taxable×rate) = %d, want 900", want)
	}
	// Invariants (single line): Σ amount == subtotal, Σ taxable == subtotal-discount.
	reconcileLines(t, inv.LineItems, inv.Subtotal, 5000, inv.TaxAmount)
}

// Task B — a billable unbilled charge is emitted as its OWN line, taxed at its
// own HSN rate, alongside the base plan line; totals reconcile.
func TestGenerateInvoice_ChargeAsOwnLine(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID: uuid.New(), Name: "SaaS", Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
		// empty HSN -> tenant SAC 998314 (18%)
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: uuid.New(), PlaceOfSupply: domain.StringPtr("TN"), // intra-state -> CGST+SGST
	}}
	ucRepo := &mockUCRepoForInvAmt{charges: []*domain.UnbilledCharge{
		{ID: uuid.New(), Amount: 50000, Currency: "INR", Description: "Real-estate fee", HSNCode: "9972"}, // 12%
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, ucRepo, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.LineItems) != 2 {
		t.Fatalf("expected 2 lines (base + charge), got %d", len(inv.LineItems))
	}
	base, charge := inv.LineItems[0], inv.LineItems[1]
	if base.Amount != 100000 || base.HSNCode != domain.DefaultSACCode || base.TaxRate != 18 {
		t.Errorf("base line = amount %d hsn %q rate %v, want 100000/998314/18", base.Amount, base.HSNCode, base.TaxRate)
	}
	if charge.Amount != 50000 || charge.HSNCode != "9972" || charge.TaxRate != 12 || charge.Description != "Real-estate fee" {
		t.Errorf("charge line = amount %d hsn %q rate %v desc %q, want 50000/9972/12/Real-estate fee",
			charge.Amount, charge.HSNCode, charge.TaxRate, charge.Description)
	}
	// base 100000 @18% = 18000; charge 50000 @12% = 6000.
	if got := base.CGSTAmount + base.SGSTAmount; got != 18000 {
		t.Errorf("base tax = %d, want 18000", got)
	}
	if got := charge.CGSTAmount + charge.SGSTAmount; got != 6000 {
		t.Errorf("charge tax = %d, want 6000", got)
	}
	if inv.Subtotal != 150000 || inv.TaxAmount != 24000 || inv.Total != 174000 {
		t.Errorf("subtotal/tax/total = %d/%d/%d, want 150000/24000/174000", inv.Subtotal, inv.TaxAmount, inv.Total)
	}
	reconcileLines(t, inv.LineItems, inv.Subtotal, 0, inv.TaxAmount)
}

// Task C — the PDF view-model builds one row per real line item (its own HSN,
// rate, and per-line CGST/SGST/IGST); legacy item-less invoices fall back to a
// single synthetic line from the totals.
func TestBuildPDFLineItems_RealAndLegacy(t *testing.T) {
	inv := &domain.Invoice{
		Currency: "INR", Subtotal: 150000, TaxAmount: 24000, IGSTAmount: 24000, Total: 174000,
		LineItems: []domain.InvoiceItem{
			{Description: "SaaS", HSNCode: "998314", Quantity: 1, UnitAmount: 100000, Amount: 100000, TaxableAmount: 100000, TaxRate: 18, IGSTAmount: 18000},
			{Description: "Real-estate", HSNCode: "9972", Quantity: 2, UnitAmount: 25000, Amount: 50000, TaxableAmount: 50000, TaxRate: 12, IGSTAmount: 6000},
		},
	}
	rows := BuildPDFLineItems(inv)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].SACCode != "998314" || rows[0].TaxRate != "18%" {
		t.Errorf("row0 HSN/rate = %q/%q, want 998314/18%%", rows[0].SACCode, rows[0].TaxRate)
	}
	if rows[1].SACCode != "9972" || rows[1].TaxRate != "12%" || rows[1].Quantity != "2" {
		t.Errorf("row1 HSN/rate/qty = %q/%q/%q, want 9972/12%%/2", rows[1].SACCode, rows[1].TaxRate, rows[1].Quantity)
	}
	if rows[0].IGSTAmount != "₹180.00" || rows[0].TaxAmount != "₹180.00" {
		t.Errorf("row0 IGST/tax = %q/%q, want ₹180.00", rows[0].IGSTAmount, rows[0].TaxAmount)
	}

	// Legacy fallback: no line items -> a single synthetic line, non-empty HSN.
	legacy := &domain.Invoice{Currency: "INR", Subtotal: 100000, TaxAmount: 18000, CGSTAmount: 9000, SGSTAmount: 9000, Total: 118000}
	legacyRows := BuildPDFLineItems(legacy)
	if len(legacyRows) != 1 {
		t.Fatalf("legacy: expected 1 synthetic row, got %d", len(legacyRows))
	}
	if legacyRows[0].SACCode == "" {
		t.Error("legacy synthetic row must carry a non-empty HSN/SAC")
	}
	if legacyRows[0].Amount != "₹1000.00" {
		t.Errorf("legacy amount = %q, want ₹1000.00", legacyRows[0].Amount)
	}
}
