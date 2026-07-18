package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Fakes for mandate metered-debit tests (Lago-parity A2) ---

// mandateMeteredInvoiceRepo records EVERY created invoice (the ceiling test
// creates two: the debit and the separate usage invoice).
type mandateMeteredInvoiceRepo struct {
	port.InvoiceRepository
	created []*domain.Invoice
}

func (m *mandateMeteredInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	m.created = append(m.created, inv)
	return nil
}

func (m *mandateMeteredInvoiceRepo) Update(ctx context.Context, inv *domain.Invoice) error {
	return nil
}

func (m *mandateMeteredInvoiceRepo) SetGatewayPaymentID(ctx context.Context, tenantID, invoiceID uuid.UUID, gatewayPaymentID string) error {
	return nil
}

type mandateMeteredSubRepo struct {
	port.SubscriptionRepository
	sub     *domain.Subscription
	updated *domain.Subscription
}

func (m *mandateMeteredSubRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return m.sub, nil
}

func (m *mandateMeteredSubRepo) Update(ctx context.Context, sub *domain.Subscription) error {
	cp := *sub
	m.updated = &cp
	return nil
}

// mandateMeteredFixture wires a MandateService whose subscription's plan has
// one per_unit charge (metric api_calls, sum) at ₹0.0035/call in INR, with a
// ₹1000.00 plan and the given usage quantity.
func mandateMeteredFixture(qty int64, maxAmount int64) (*MandateService, *mandateMeteredInvoiceRepo, *mandateMeteredSubRepo, *mockRatingRepoForMeter, *mandateMockGateway, *domain.Mandate) {
	subID := uuid.New()
	mandate := newTestMandate()
	mandate.SubscriptionID = &subID
	mandate.MaxAmount = maxAmount

	sub := &domain.Subscription{
		ID:                 subID,
		TenantID:           mandate.TenantID,
		CustomerID:         mandate.CustomerID,
		PlanID:             uuid.New(),
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	subRepo := &mandateMeteredSubRepo{sub: sub}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:            sub.PlanID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 100000, Currency: "INR", Type: "recurring"}},
	}}
	invRepo := &mandateMeteredInvoiceRepo{}
	gw := &mandateMockGateway{debitResult: &port.PaymentResult{Success: true, PaymentID: "pay_meter1"}}
	custRepo := &mandateMockCustomerRepo{customer: &domain.Customer{
		ID:            mandate.CustomerID,
		Email:         "c@example.com",
		PlaceOfSupply: domain.StringPtr("KA"), // inter-state -> IGST
	}}

	svc := NewMandateService(&mandateMockRepo{mandate: mandate}, gw, custRepo, invRepo)
	svc.SetBillingResolver(subRepo, planRepo, NewTaxResolver(nil, "", ""))

	// The InvoiceService supplies metered rating; it shares the same repos.
	metricID, chargeID := uuid.New(), uuid.New()
	metric := domain.BillableMetric{ID: metricID, Name: "API calls", Code: "api_calls", AggregationType: domain.AggregationSum}
	invoiceSvc := NewInvoiceService(invRepo, planRepo, &mockCustomerRepoForInvAmt{customer: custRepo.customer}, &mockUCRepoForInvAmt{}, subRepo, nil, nil)
	invoiceSvc.ChargeRepo = &mockChargeRepoForMeter{charges: []domain.Charge{{
		ID:          chargeID,
		PlanID:      sub.PlanID,
		MetricID:    metricID,
		ChargeModel: domain.ChargePerUnit,
		Amounts:     map[string]domain.ChargeAmounts{"INR": {UnitAmount: "0.0035"}},
		Metric:      &metric,
	}}}
	invoiceSvc.UsageRepo = &mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": qty}}
	ratingRepo := &mockRatingRepoForMeter{rated: map[uuid.UUID]bool{}}
	invoiceSvc.RatingRepo = ratingRepo
	svc.SetInvoiceService(invoiceSvc)

	return svc, invRepo, subRepo, ratingRepo, gw, mandate
}

func TestDebitSubscription_MeteredLinesOnDebitInvoice(t *testing.T) {
	// 1500 × ₹0.0035 = 525p usage; base 100000p; generous ceiling.
	svc, invRepo, subRepo, ratingRepo, gw, mandate := mandateMeteredFixture(1500, 1_000_000)

	if err := svc.DebitSubscription(context.Background(), mandate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(invRepo.created) != 1 {
		t.Fatalf("invoices created = %d, want 1", len(invRepo.created))
	}
	inv := invRepo.created[0]
	if len(inv.LineItems) != 2 {
		t.Fatalf("line count = %d, want 2 (debit + metered)", len(inv.LineItems))
	}
	if inv.Subtotal != 100525 {
		t.Fatalf("Subtotal = %d, want 100525", inv.Subtotal)
	}
	if inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Fatalf("Total (%d) != Subtotal (%d) + Tax (%d)", inv.Total, inv.Subtotal, inv.TaxAmount)
	}
	var lineSum, lineTax int64
	for _, li := range inv.LineItems {
		lineSum += li.Amount
		lineTax += li.IGSTAmount + li.CGSTAmount + li.SGSTAmount
	}
	if lineSum != inv.Subtotal || lineTax != inv.TaxAmount {
		t.Fatalf("Σ lines %d/%d, want %d/%d (exact reconciliation)", lineSum, lineTax, inv.Subtotal, inv.TaxAmount)
	}
	if !strings.Contains(inv.LineItems[1].Description, "API calls") {
		t.Fatalf("metered line description %q missing metric name", inv.LineItems[1].Description)
	}

	// The gateway is charged the FULL total (base + usage + tax).
	if gw.debitCalls != 1 || gw.lastDebitAmount != inv.Total {
		t.Fatalf("gateway charged %d× for %d, want once for %d", gw.debitCalls, gw.lastDebitAmount, inv.Total)
	}

	// The usage window is claimed and the subscription period advanced.
	if len(ratingRepo.created) != 1 || ratingRepo.created[0].InvoiceID != inv.ID {
		t.Fatalf("rating claims = %+v, want one on invoice %s", ratingRepo.created, inv.ID)
	}
	if subRepo.updated == nil {
		t.Fatal("subscription period must advance on mandate debit")
	}
	if !subRepo.updated.CurrentPeriodStart.Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("period start = %v, want old end 2026-07-01", subRepo.updated.CurrentPeriodStart)
	}
	if !subRepo.updated.CurrentPeriodEnd.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("period end = %v, want 2026-08-01", subRepo.updated.CurrentPeriodEnd)
	}
}

func TestDebitSubscription_UsageOverCeilingBilledSeparately(t *testing.T) {
	// Base = 100000 + 18% IGST = 118000. Ceiling 120000: base fits, but usage
	// (1,000,000 calls × ₹0.0035 = 350000p + tax) does not.
	svc, invRepo, _, ratingRepo, gw, mandate := mandateMeteredFixture(1_000_000, 120000)

	if err := svc.DebitSubscription(context.Background(), mandate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two invoices: the separate usage-only invoice, then the base debit.
	if len(invRepo.created) != 2 {
		t.Fatalf("invoices created = %d, want 2 (usage + debit)", len(invRepo.created))
	}
	var debit, usage *domain.Invoice
	for _, inv := range invRepo.created {
		if inv.BillingReason == domain.BillingReasonMandateDebit {
			debit = inv
		} else {
			usage = inv
		}
	}
	if debit == nil || usage == nil {
		t.Fatalf("expected one mandate-debit and one usage invoice, got %+v", invRepo.created)
	}
	if debit.Subtotal != 100000 || len(debit.LineItems) != 1 {
		t.Fatalf("debit invoice subtotal/lines = %d/%d, want 100000/1 (flat only)", debit.Subtotal, len(debit.LineItems))
	}
	if gw.lastDebitAmount != debit.Total {
		t.Fatalf("gateway charged %d, want the flat-only total %d (never exceed the mandate ceiling)", gw.lastDebitAmount, debit.Total)
	}
	if debit.Total > mandate.MaxAmount {
		t.Fatalf("debit total %d exceeds authorized max %d", debit.Total, mandate.MaxAmount)
	}
	if usage.Subtotal != 350000 || len(usage.LineItems) != 1 {
		t.Fatalf("usage invoice subtotal/lines = %d/%d, want 350000/1", usage.Subtotal, len(usage.LineItems))
	}
	// The separate usage invoice claims the window, so the next debit can't re-bill it.
	if len(ratingRepo.created) != 1 || ratingRepo.created[0].InvoiceID != usage.ID {
		t.Fatalf("rating claims = %+v, want one on the usage invoice %s", ratingRepo.created, usage.ID)
	}
}

func TestDebitSubscription_NoInvoiceServiceStaysFlatOnly(t *testing.T) {
	svc, invRepo, _, _, _, mandate := mandateMeteredFixture(1500, 1_000_000)
	svc.SetInvoiceService(nil) // metering not wired

	if err := svc.DebitSubscription(context.Background(), mandate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.created) != 1 || len(invRepo.created[0].LineItems) != 1 {
		t.Fatalf("expected the pre-A2 single-line debit invoice, got %+v", invRepo.created)
	}
}
