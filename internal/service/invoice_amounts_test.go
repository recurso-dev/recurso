package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/gsp"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Mocks for invoice amount tests ---

type mockInvoiceRepoForInvAmt struct {
	port.InvoiceRepository
	created   *domain.Invoice
	createErr error
}

func (m *mockInvoiceRepoForInvAmt) Create(ctx context.Context, inv *domain.Invoice) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = inv
	return nil
}

type mockPlanRepoForInvAmt struct {
	port.PlanRepository
	plan *domain.Plan
}

func (m *mockPlanRepoForInvAmt) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return m.plan, nil
}

type mockCustomerRepoForInvAmt struct {
	port.CustomerRepository
	customer *domain.Customer
}

func (m *mockCustomerRepoForInvAmt) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.customer, nil
}

type mockUCRepoForInvAmt struct {
	port.UnbilledChargeRepository
	charges     []*domain.UnbilledCharge
	listErr     error
	invoicedIDs []uuid.UUID
}

func (m *mockUCRepoForInvAmt) ListBySubscriptionID(subID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.charges, nil
}

func (m *mockUCRepoForInvAmt) MarkAsInvoiced(ids []uuid.UUID) error {
	m.invoicedIDs = append(m.invoicedIDs, ids...)
	return nil
}

type mockSubRepoForInvAmt struct {
	port.SubscriptionRepository
	sub     *domain.Subscription
	updated *domain.Subscription
}

func (m *mockSubRepoForInvAmt) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return m.sub, nil
}

func (m *mockSubRepoForInvAmt) Update(ctx context.Context, sub *domain.Subscription) error {
	m.updated = sub
	return nil
}

func newInvAmtService(
	invRepo *mockInvoiceRepoForInvAmt,
	planRepo *mockPlanRepoForInvAmt,
	custRepo *mockCustomerRepoForInvAmt,
	ucRepo *mockUCRepoForInvAmt,
	subRepo *mockSubRepoForInvAmt,
) *InvoiceService {
	// nil resolver -> env-default IN/TN, matching the historical behavior
	// these tests assert.
	return NewInvoiceService(invRepo, planRepo, custRepo, ucRepo, subRepo, gsp.NewMockGSPAdapter(), nil)
}

// --- GenerateInvoice arithmetic tests ---

func TestGenerateInvoice_UnbilledChargesAddedToSubtotal(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"), // inter-state vs org "TN" -> IGST
	}}
	chargeID1, chargeID2 := uuid.New(), uuid.New()
	ucRepo := &mockUCRepoForInvAmt{charges: []*domain.UnbilledCharge{
		{ID: chargeID1, Amount: 5000},
		{ID: chargeID2, Amount: 2500},
	}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, ucRepo, &mockSubRepoForInvAmt{})

	sub := &domain.Subscription{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	}

	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subtotal = plan price + unbilled charges = 100000 + 5000 + 2500
	if inv.Subtotal != 107500 {
		t.Errorf("Subtotal = %d, want 107500", inv.Subtotal)
	}
	// Inter-state 18% IGST on 107500 = 19350
	if inv.TaxAmount != 19350 {
		t.Errorf("TaxAmount = %d, want 19350", inv.TaxAmount)
	}
	if inv.IGSTAmount != 19350 {
		t.Errorf("IGSTAmount = %d, want 19350", inv.IGSTAmount)
	}
	if inv.CGSTAmount != 0 || inv.SGSTAmount != 0 {
		t.Errorf("CGST/SGST = %d/%d, want 0/0 for inter-state", inv.CGSTAmount, inv.SGSTAmount)
	}
	if inv.Total != 126850 {
		t.Errorf("Total = %d, want 126850 (107500 + 19350)", inv.Total)
	}
	if inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Errorf("Total (%d) != Subtotal (%d) + TaxAmount (%d)", inv.Total, inv.Subtotal, inv.TaxAmount)
	}
	if inv.Status != domain.InvoiceStatusOpen {
		t.Errorf("Status = %q, want %q", inv.Status, domain.InvoiceStatusOpen)
	}
	if inv.Currency != "INR" {
		t.Errorf("Currency = %q, want INR", inv.Currency)
	}
	// Persisted invoice is the same one returned
	if invRepo.created != inv {
		t.Error("invoice returned differs from invoice persisted")
	}
	// Charges marked as invoiced
	if len(ucRepo.invoicedIDs) != 2 {
		t.Fatalf("expected 2 charges marked invoiced, got %d", len(ucRepo.invoicedIDs))
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range ucRepo.invoicedIDs {
		seen[id] = true
	}
	if !seen[chargeID1] || !seen[chargeID2] {
		t.Errorf("invoiced IDs %v missing expected charge IDs", ucRepo.invoicedIDs)
	}
}

func TestGenerateInvoice_UnbilledChargeListError_ChargesSkipped(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"),
	}}
	ucRepo := &mockUCRepoForInvAmt{listErr: errors.New("db down")}

	svc := newInvAmtService(invRepo, planRepo, custRepo, ucRepo, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Charge listing error is swallowed; subtotal falls back to plan price only.
	if inv.Subtotal != 100000 {
		t.Errorf("Subtotal = %d, want 100000 (plan price only)", inv.Subtotal)
	}
	if len(ucRepo.invoicedIDs) != 0 {
		t.Errorf("expected no charges marked invoiced, got %d", len(ucRepo.invoicedIDs))
	}
}

func TestGenerateInvoice_PlanWithoutPrices_Error(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{ID: uuid.New()}} // no prices
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: uuid.New()}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	_, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	})
	if err == nil {
		t.Fatal("expected error for plan with no prices")
	}
	if invRepo.created != nil {
		t.Error("no invoice should be persisted when plan has no prices")
	}
}

func TestGenerateInvoice_RepoCreateError_Propagated(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{createErr: errors.New("insert failed")}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: uuid.New()}}
	ucRepo := &mockUCRepoForInvAmt{charges: []*domain.UnbilledCharge{{ID: uuid.New(), Amount: 100}}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, ucRepo, &mockSubRepoForInvAmt{})

	_, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	})
	if err == nil {
		t.Fatal("expected error when invoice persistence fails")
	}
	// Charges must NOT be marked invoiced when the invoice was never persisted.
	if len(ucRepo.invoicedIDs) != 0 {
		t.Errorf("expected 0 charges marked invoiced after create failure, got %d", len(ucRepo.invoicedIDs))
	}
}

func TestGenerateInvoice_NonINRCurrencyPropagated(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 9900, Currency: "USD"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:             uuid.New(),
		BillingAddress: domain.BillingAddress{Country: "US"},
	}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", inv.Currency)
	}
	if inv.Subtotal != 9900 {
		t.Errorf("Subtotal = %d, want 9900", inv.Subtotal)
	}
	// The arithmetic invariant must hold regardless of currency.
	if inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Errorf("Total (%d) != Subtotal (%d) + TaxAmount (%d)", inv.Total, inv.Subtotal, inv.TaxAmount)
	}
	// India GST must not apply to non-INR invoices.
	if inv.TaxAmount != 0 {
		t.Errorf("TaxAmount = %d, want 0 (GST must not apply to USD invoices)", inv.TaxAmount)
	}
	if inv.IGSTAmount != 0 || inv.CGSTAmount != 0 || inv.SGSTAmount != 0 {
		t.Errorf("GST components = IGST %d / CGST %d / SGST %d, want all 0 for USD",
			inv.IGSTAmount, inv.CGSTAmount, inv.SGSTAmount)
	}
	if inv.Total != 9900 {
		t.Errorf("Total = %d, want 9900", inv.Total)
	}
}

func TestGenerateInvoice_MismatchedCurrencyChargesSkipped(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"),
	}}
	inrCharge, usdCharge := uuid.New(), uuid.New()
	ucRepo := &mockUCRepoForInvAmt{charges: []*domain.UnbilledCharge{
		{ID: inrCharge, Amount: 5000, Currency: "INR"},
		{ID: usdCharge, Amount: 9900, Currency: "USD"}, // must not be summed into an INR invoice
	}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, ucRepo, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.Subtotal != 105000 {
		t.Errorf("Subtotal = %d, want 105000 (plan price + INR charge only)", inv.Subtotal)
	}
	// Only the matching-currency charge is marked invoiced; the USD charge
	// stays unbilled for a future USD invoice.
	if len(ucRepo.invoicedIDs) != 1 || ucRepo.invoicedIDs[0] != inrCharge {
		t.Errorf("invoiced IDs = %v, want exactly [%v]", ucRepo.invoicedIDs, inrCharge)
	}
}

func TestGenerateInvoice_DefaultPaymentTermsNet0(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 1000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: uuid.New()}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID:         uuid.New(),
		CustomerID: custRepo.customer.ID,
		PlanID:     planRepo.plan.ID,
		// PaymentTerms empty
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.PaymentTerms != "net0" {
		t.Errorf("PaymentTerms = %q, want net0 default", inv.PaymentTerms)
	}
}

// --- GenerateAdvanceInvoice tests ---

func TestGenerateAdvanceInvoice_AmountsAndPeriodExtension(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	periodEnd := time.Now().UTC().Add(10 * 24 * time.Hour).Truncate(time.Second)

	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 10000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            customerID,
		PlaceOfSupply: domain.StringPtr("TN"), // intra-state -> CGST+SGST
	}}
	subRepo := &mockSubRepoForInvAmt{sub: &domain.Subscription{
		ID:               subID,
		TenantID:         uuid.New(),
		CustomerID:       customerID,
		PlanID:           planID,
		CurrentPeriodEnd: periodEnd,
	}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, subRepo)

	inv, err := svc.GenerateAdvanceInvoice(context.Background(), subID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 periods x 10000 = 30000 subtotal
	if inv.Subtotal != 30000 {
		t.Errorf("Subtotal = %d, want 30000", inv.Subtotal)
	}
	// Intra-state 18% on 30000 = 5400 (2700 CGST + 2700 SGST)
	if inv.TaxAmount != 5400 {
		t.Errorf("TaxAmount = %d, want 5400", inv.TaxAmount)
	}
	if inv.CGSTAmount != 2700 || inv.SGSTAmount != 2700 {
		t.Errorf("CGST/SGST = %d/%d, want 2700/2700", inv.CGSTAmount, inv.SGSTAmount)
	}
	if inv.IGSTAmount != 0 {
		t.Errorf("IGSTAmount = %d, want 0", inv.IGSTAmount)
	}
	if inv.Total != 35400 {
		t.Errorf("Total = %d, want 35400", inv.Total)
	}

	// Subscription period extended by 3 monthly intervals from the current end.
	if subRepo.updated == nil {
		t.Fatal("expected subscription to be updated with extended period")
	}
	want := periodEnd
	for i := 0; i < 3; i++ {
		want = domain.AddInterval(want, string(domain.IntervalMonth), 1)
	}
	if !subRepo.updated.CurrentPeriodEnd.Equal(want) {
		t.Errorf("CurrentPeriodEnd = %v, want %v", subRepo.updated.CurrentPeriodEnd, want)
	}
}

func TestGenerateAdvanceInvoice_ExpiredPeriodExtendsFromNow(t *testing.T) {
	planID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	// Period already ended in the past
	pastEnd := time.Now().UTC().Add(-30 * 24 * time.Hour)

	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:            planID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 5000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: customerID}}
	subRepo := &mockSubRepoForInvAmt{sub: &domain.Subscription{
		ID:               subID,
		CustomerID:       customerID,
		PlanID:           planID,
		CurrentPeriodEnd: pastEnd,
	}}

	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, subRepo)

	before := time.Now()
	_, err := svc.GenerateAdvanceInvoice(context.Background(), subID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if subRepo.updated == nil {
		t.Fatal("expected subscription update")
	}
	// New end must be anchored at ~now (not the stale past end), plus 1 month.
	lowerBound := domain.AddInterval(before, string(domain.IntervalMonth), 1)
	if subRepo.updated.CurrentPeriodEnd.Before(lowerBound.Add(-time.Minute)) {
		t.Errorf("CurrentPeriodEnd = %v, want >= ~%v (anchored at now)", subRepo.updated.CurrentPeriodEnd, lowerBound)
	}
}
