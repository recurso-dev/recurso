package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Fakes for the renewal engine (Lago-parity A1) ---

type fakeRenewalSubRepo struct {
	claimed  []*domain.Subscription
	claimErr error
	updated  []*domain.Subscription
	updErr   error
}

func (f *fakeRenewalSubRepo) ClaimDueForRenewal(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	return f.claimed, f.claimErr
}

func (f *fakeRenewalSubRepo) Update(ctx context.Context, sub *domain.Subscription) error {
	if f.updErr != nil {
		return f.updErr
	}
	cp := *sub
	f.updated = append(f.updated, &cp)
	return nil
}

type fakeRenewalInvoicer struct {
	invoice    *domain.Invoice
	invErr     error
	finalInv   *domain.Invoice
	finalErr   error
	generated  int
	finalCalls []time.Time
	seenPeriod [2]time.Time
}

func (f *fakeRenewalInvoicer) GenerateInvoice(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	f.generated++
	f.seenPeriod = [2]time.Time{sub.CurrentPeriodStart, sub.CurrentPeriodEnd}
	return f.invoice, f.invErr
}

func (f *fakeRenewalInvoicer) GenerateFinalUsageInvoice(ctx context.Context, sub *domain.Subscription, endedAt time.Time) (*domain.Invoice, error) {
	f.finalCalls = append(f.finalCalls, endedAt)
	return f.finalInv, f.finalErr
}

type fakeRenewalCharger struct {
	result  *port.PaymentResult
	err     error
	charges int
	amount  int64
	idemKey string
}

func (f *fakeRenewalCharger) ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error) {
	f.charges++
	f.amount = amount
	f.idemKey = idempotencyKey
	return f.result, f.err
}

type fakeRenewalLookup struct {
	stripeID, methodID string
}

func (f *fakeRenewalLookup) GetSavedPaymentMethod(ctx context.Context, customerID uuid.UUID) (string, string, error) {
	return f.stripeID, f.methodID, nil
}

type fakeRenewalSettler struct{ settled []uuid.UUID }

func (f *fakeRenewalSettler) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (bool, error) {
	f.settled = append(f.settled, invoiceID)
	return true, nil
}

func renewalFixture() (*RenewalService, *fakeRenewalSubRepo, *fakeRenewalInvoicer, *domain.Subscription) {
	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		CustomerID:         uuid.New(),
		PlanID:             uuid.New(),
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	subRepo := &fakeRenewalSubRepo{claimed: []*domain.Subscription{sub}}
	invoicer := &fakeRenewalInvoicer{invoice: &domain.Invoice{
		ID:         uuid.New(),
		CustomerID: sub.CustomerID,
		Status:     domain.InvoiceStatusOpen,
		Currency:   "INR",
		Total:      118000,
	}}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:            sub.PlanID,
		IntervalUnit:  domain.IntervalMonth,
		IntervalCount: 1,
		Prices:        []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	svc := NewRenewalService(subRepo, planRepo, invoicer)
	svc.now = func() time.Time { return time.Date(2026, 7, 1, 0, 5, 0, 0, time.UTC) }
	return svc, subRepo, invoicer, sub
}

func TestRenewSubscription_InvoicesThenAdvancesPeriod(t *testing.T) {
	svc, subRepo, invoicer, sub := renewalFixture()
	oldStart, oldEnd := sub.CurrentPeriodStart, sub.CurrentPeriodEnd

	if err := svc.RenewSubscription(context.Background(), sub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The invoice must see the ELAPSED period (usage rated in arrears).
	if invoicer.generated != 1 {
		t.Fatalf("GenerateInvoice calls = %d, want 1", invoicer.generated)
	}
	if !invoicer.seenPeriod[0].Equal(oldStart) || !invoicer.seenPeriod[1].Equal(oldEnd) {
		t.Fatalf("invoice saw period %v–%v, want the un-advanced %v–%v",
			invoicer.seenPeriod[0], invoicer.seenPeriod[1], oldStart, oldEnd)
	}

	// Then the period advances: start = old end, end = +1 month.
	if len(subRepo.updated) != 1 {
		t.Fatalf("subscription updates = %d, want 1", len(subRepo.updated))
	}
	got := subRepo.updated[0]
	if !got.CurrentPeriodStart.Equal(oldEnd) {
		t.Fatalf("new period start = %v, want old end %v", got.CurrentPeriodStart, oldEnd)
	}
	wantEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if !got.CurrentPeriodEnd.Equal(wantEnd) {
		t.Fatalf("new period end = %v, want %v", got.CurrentPeriodEnd, wantEnd)
	}
	if got.Status != domain.SubscriptionStatusActive {
		t.Fatalf("status = %q, want active", got.Status)
	}
}

func TestRenewSubscription_CancelAtPeriodEnd(t *testing.T) {
	svc, subRepo, invoicer, sub := renewalFixture()
	sub.CancelAtPeriodEnd = true
	periodEnd := sub.CurrentPeriodEnd

	if err := svc.RenewSubscription(context.Background(), sub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Final usage invoice for the elapsed window, NO renewal invoice.
	if invoicer.generated != 0 {
		t.Fatalf("GenerateInvoice calls = %d, want 0 for period-end cancel", invoicer.generated)
	}
	if len(invoicer.finalCalls) != 1 || !invoicer.finalCalls[0].Equal(periodEnd) {
		t.Fatalf("final invoice calls = %v, want one at %v", invoicer.finalCalls, periodEnd)
	}

	if len(subRepo.updated) != 1 {
		t.Fatalf("subscription updates = %d, want 1", len(subRepo.updated))
	}
	got := subRepo.updated[0]
	if got.Status != domain.SubscriptionStatusCanceled || got.CanceledAt == nil || got.CancelAtPeriodEnd {
		t.Fatalf("expected canceled with CanceledAt set and flag cleared, got %+v", got)
	}
	// The period must NOT advance — service ended at period end.
	if !got.CurrentPeriodEnd.Equal(periodEnd) {
		t.Fatalf("period end moved to %v on cancellation, want unchanged %v", got.CurrentPeriodEnd, periodEnd)
	}
}

func TestRenewSubscription_InvoiceFailureLeavesPeriodUntouched(t *testing.T) {
	svc, subRepo, invoicer, sub := renewalFixture()
	invoicer.invErr = errors.New("db down")

	if err := svc.RenewSubscription(context.Background(), sub); err == nil {
		t.Fatal("expected error when invoice generation fails")
	}
	if len(subRepo.updated) != 0 {
		t.Fatal("period must not advance when the invoice failed (lease retries)")
	}
}

func TestRenewSubscription_PaymentSuccessSettles(t *testing.T) {
	svc, _, invoicer, sub := renewalFixture()
	charger := &fakeRenewalCharger{result: &port.PaymentResult{Success: true}}
	settler := &fakeRenewalSettler{}
	svc.SetSavedMethodCharging(charger, &fakeRenewalLookup{stripeID: "cus_1", methodID: "pm_1"}, settler)

	if err := svc.RenewSubscription(context.Background(), sub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charger.charges != 1 || charger.amount != invoicer.invoice.Total {
		t.Fatalf("charged %d times for %d, want once for %d", charger.charges, charger.amount, invoicer.invoice.Total)
	}
	// Idempotency key is invoice-derived so a crash re-run can't double-charge.
	if want := "renewal-" + invoicer.invoice.ID.String(); charger.idemKey != want {
		t.Fatalf("idempotency key = %q, want %q", charger.idemKey, want)
	}
	if len(settler.settled) != 1 || settler.settled[0] != invoicer.invoice.ID {
		t.Fatalf("settled = %v, want [%v]", settler.settled, invoicer.invoice.ID)
	}
}

func TestRenewSubscription_PaymentDeclineLeavesInvoiceOpen(t *testing.T) {
	svc, subRepo, _, sub := renewalFixture()
	charger := &fakeRenewalCharger{result: &port.PaymentResult{Success: false, ErrorCode: "insufficient_funds"}}
	settler := &fakeRenewalSettler{}
	svc.SetSavedMethodCharging(charger, &fakeRenewalLookup{stripeID: "cus_1", methodID: "pm_1"}, settler)

	if err := svc.RenewSubscription(context.Background(), sub); err != nil {
		t.Fatalf("a payment decline must not fail the renewal: %v", err)
	}
	if len(settler.settled) != 0 {
		t.Fatal("declined charge must not settle the invoice")
	}
	if len(subRepo.updated) != 1 {
		t.Fatal("period must still advance on payment decline (dunning owns recovery)")
	}
}

func TestRenewSubscription_NoSavedMethodNoCharge(t *testing.T) {
	svc, _, _, sub := renewalFixture()
	charger := &fakeRenewalCharger{}
	svc.SetSavedMethodCharging(charger, &fakeRenewalLookup{}, &fakeRenewalSettler{}) // empty lookup

	if err := svc.RenewSubscription(context.Background(), sub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charger.charges != 0 {
		t.Fatal("no saved method: gateway must not be called")
	}
}

func TestProcessDueRenewals_ContinuesPastFailures(t *testing.T) {
	svc, subRepo, invoicer, sub := renewalFixture()
	bad := *sub
	bad.ID = uuid.New()
	bad.PlanID = sub.PlanID
	subRepo.claimed = []*domain.Subscription{&bad, sub}

	// First renewal fails at invoice time, second succeeds.
	calls := 0
	invoicer.invErr = nil
	origInvoice := invoicer.invoice
	failer := &flakyInvoicer{inner: invoicer, failFirst: &calls, invoice: origInvoice}
	svc.invoicer = failer

	renewed, err := svc.ProcessDueRenewals(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renewed != 1 {
		t.Fatalf("renewed = %d, want 1 (one failure skipped, one success)", renewed)
	}
}

type flakyInvoicer struct {
	inner     *fakeRenewalInvoicer
	failFirst *int
	invoice   *domain.Invoice
}

func (f *flakyInvoicer) GenerateInvoice(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	*f.failFirst++
	if *f.failFirst == 1 {
		return nil, errors.New("transient failure")
	}
	return f.invoice, nil
}

func (f *flakyInvoicer) GenerateFinalUsageInvoice(ctx context.Context, sub *domain.Subscription, endedAt time.Time) (*domain.Invoice, error) {
	return f.inner.GenerateFinalUsageInvoice(ctx, sub, endedAt)
}

func TestProcessDueRenewals_ClaimErrorAborts(t *testing.T) {
	svc, subRepo, _, _ := renewalFixture()
	subRepo.claimErr = errors.New("db down")
	if _, err := svc.ProcessDueRenewals(context.Background()); err == nil {
		t.Fatal("expected claim error to abort the sweep")
	}
}
