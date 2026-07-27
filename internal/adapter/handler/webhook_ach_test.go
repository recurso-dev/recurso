package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/stripe/stripe-go/v76"
)

// memAttempts is an in-memory paymentAttemptStore keyed by PaymentIntent id.
type memAttempts struct {
	byPI map[string]*domain.PaymentAttempt
}

func newMemAttempts() *memAttempts { return &memAttempts{byPI: map[string]*domain.PaymentAttempt{}} }

func (m *memAttempts) Create(_ context.Context, a *domain.PaymentAttempt) error {
	cp := *a
	m.byPI[a.GatewayPaymentIntentID] = &cp
	return nil
}
func (m *memAttempts) GetByPaymentIntentID(_ context.Context, pi string) (*domain.PaymentAttempt, error) {
	return m.byPI[pi], nil
}
func (m *memAttempts) UpdateStatusByPaymentIntent(_ context.Context, pi string, status domain.PaymentAttemptStatus, code string, settled *time.Time) error {
	a := m.byPI[pi]
	if a == nil {
		return fmt.Errorf("no attempt for %s", pi)
	}
	a.Status = status
	a.FailureCode = code
	a.SettledAt = settled
	return nil
}

// achInvoiceRepo stubs only GetByIDPublic (the rest of the port is unused here).
type achInvoiceRepo struct {
	port.InvoiceRepository
	inv *domain.Invoice
}

func (r *achInvoiceRepo) GetByIDPublic(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}

func piEvent(pi stripe.PaymentIntent) stripe.Event {
	raw, _ := json.Marshal(pi)
	return stripe.Event{Data: &stripe.EventData{Raw: raw}}
}

// payment_intent.processing on an ACH debit records a processing attempt bound
// to the invoice — the in-flight state the invoice enum can't express.
func TestACHWebhook_ProcessingCreatesAttempt(t *testing.T) {
	tenantID, invoiceID := uuid.New(), uuid.New()
	attempts := newMemAttempts()
	h := &WebhookHandler{
		logger:          slog.Default(),
		paymentAttempts: attempts,
		invoiceRepo:     &achInvoiceRepo{inv: &domain.Invoice{ID: invoiceID, TenantID: tenantID}},
	}
	pi := stripe.PaymentIntent{
		ID: "pi_ach_1", Amount: 108750,
		PaymentMethodTypes: []string{"us_bank_account"},
		Metadata:           map[string]string{"invoice_id": invoiceID.String()},
	}
	if err := h.handlePaymentIntentProcessing(context.Background(), piEvent(pi)); err != nil {
		t.Fatalf("processing: %v", err)
	}
	a := attempts.byPI["pi_ach_1"]
	if a == nil || a.Status != domain.PaymentAttemptProcessing || a.Method != "us_bank_account" ||
		a.Amount != 108750 || a.InvoiceID != invoiceID || a.TenantID != tenantID {
		t.Fatalf("attempt wrong: %+v", a)
	}
	if !a.InFlight() {
		t.Error("a processing attempt must read as in-flight")
	}
}

// A redelivered processing event advances the same attempt, not a duplicate.
func TestACHWebhook_ProcessingIsIdempotent(t *testing.T) {
	invoiceID := uuid.New()
	attempts := newMemAttempts()
	attempts.byPI["pi_ach_1"] = &domain.PaymentAttempt{GatewayPaymentIntentID: "pi_ach_1", Status: domain.PaymentAttemptInitiated}
	h := &WebhookHandler{
		logger:          slog.Default(),
		paymentAttempts: attempts,
		invoiceRepo:     &achInvoiceRepo{inv: &domain.Invoice{ID: invoiceID}},
	}
	pi := stripe.PaymentIntent{ID: "pi_ach_1", Metadata: map[string]string{"invoice_id": invoiceID.String()}}
	if err := h.handlePaymentIntentProcessing(context.Background(), piEvent(pi)); err != nil {
		t.Fatalf("processing: %v", err)
	}
	if len(attempts.byPI) != 1 || attempts.byPI["pi_ach_1"].Status != domain.PaymentAttemptProcessing {
		t.Fatalf("expected single advanced attempt, got %+v", attempts.byPI)
	}
}

// payment_intent.succeeded settles the attempt.
func TestACHWebhook_SucceededSettles(t *testing.T) {
	attempts := newMemAttempts()
	attempts.byPI["pi_ach_2"] = &domain.PaymentAttempt{GatewayPaymentIntentID: "pi_ach_2", Status: domain.PaymentAttemptProcessing}
	h := &WebhookHandler{logger: slog.Default(), paymentAttempts: attempts}
	h.settleAttempt(context.Background(), "pi_ach_2")
	a := attempts.byPI["pi_ach_2"]
	if a.Status != domain.PaymentAttemptSucceeded || a.SettledAt == nil {
		t.Fatalf("expected settled, got %+v", a)
	}
}

// payment_intent.payment_failed records the failure with its ACH code.
func TestACHWebhook_PaymentFailedRecordsCode(t *testing.T) {
	attempts := newMemAttempts()
	attempts.byPI["pi_ach_3"] = &domain.PaymentAttempt{GatewayPaymentIntentID: "pi_ach_3", Status: domain.PaymentAttemptProcessing}
	h := &WebhookHandler{logger: slog.Default(), paymentAttempts: attempts}
	pi := stripe.PaymentIntent{ID: "pi_ach_3", LastPaymentError: &stripe.Error{Code: "insufficient_funds"}}
	if err := h.handlePaymentIntentPaymentFailed(context.Background(), piEvent(pi)); err != nil {
		t.Fatalf("payment_failed: %v", err)
	}
	a := attempts.byPI["pi_ach_3"]
	if a.Status != domain.PaymentAttemptFailed || a.FailureCode != "insufficient_funds" || a.InFlight() {
		t.Fatalf("expected failed+code, got %+v", a)
	}
}

// With no store wired, the new handlers are safe no-ops (existing deploys).
func TestACHWebhook_NilStoreIsNoop(t *testing.T) {
	h := &WebhookHandler{logger: slog.Default()}
	pi := stripe.PaymentIntent{ID: "pi_x", Metadata: map[string]string{"invoice_id": uuid.New().String()}}
	if err := h.handlePaymentIntentProcessing(context.Background(), piEvent(pi)); err != nil {
		t.Fatalf("nil store must be a no-op (processing): %v", err)
	}
	if err := h.handlePaymentIntentPaymentFailed(context.Background(), piEvent(pi)); err != nil {
		t.Fatalf("nil store must be a no-op (failed): %v", err)
	}
	h.settleAttempt(context.Background(), "pi_x") // must not panic
}

// --- Inc 3c: late ACH return reverses the settlement ---

// returnStubRepo stubs the two reads ReverseSettledPayment needs so the webhook
// test can drive the whole return chain (mark attempt returned → reopen invoice)
// in memory. The ledger is left nil on the SubscriptionService (nil-safe).
type returnStubRepo struct {
	port.InvoiceRepository
	inv          *domain.Invoice
	reverseCalls int
}

func (r *returnStubRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}
func (r *returnStubRepo) ReverseToUnpaid(_ context.Context, _, _ uuid.UUID) (bool, error) {
	r.reverseCalls++
	r.inv.Status = domain.InvoiceStatusPastDue // reflect the transition for redelivery
	return true, nil
}

func refundEvt(piID string, status stripe.RefundStatus) *stripe.Refund {
	return &stripe.Refund{
		ID:            "re_" + piID,
		Status:        status,
		PaymentIntent: &stripe.PaymentIntent{ID: piID},
	}
}

// A succeeded refund that owns no credit note, tied to a settled us_bank_account
// attempt, is an involuntary ACH return: the attempt is marked `returned` and
// the invoice is reopened (paid → past_due) for re-collection.
func TestACHWebhook_ReturnMarksAttemptAndReopensInvoice(t *testing.T) {
	tenantID, invoiceID := uuid.New(), uuid.New()
	attempts := newMemAttempts()
	attempts.byPI["pi_ret"] = &domain.PaymentAttempt{
		GatewayPaymentIntentID: "pi_ret", Status: domain.PaymentAttemptSucceeded,
		Method: "us_bank_account", InvoiceID: invoiceID, TenantID: tenantID,
	}
	repo := &returnStubRepo{inv: &domain.Invoice{
		ID: invoiceID, TenantID: tenantID, Status: domain.InvoiceStatusPaid, Total: 1000,
	}}
	subSvc := service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := &WebhookHandler{logger: slog.Default(), paymentAttempts: attempts, subService: subSvc}

	if err := h.handleACHReturn(context.Background(), refundEvt("pi_ret", stripe.RefundStatusSucceeded)); err != nil {
		t.Fatalf("handleACHReturn: %v", err)
	}
	if got := attempts.byPI["pi_ret"].Status; got != domain.PaymentAttemptReturned {
		t.Errorf("attempt status = %s, want returned", got)
	}
	if repo.reverseCalls != 1 {
		t.Errorf("ReverseToUnpaid called %d times, want 1", repo.reverseCalls)
	}
	if repo.inv.Status != domain.InvoiceStatusPastDue {
		t.Errorf("invoice status = %s, want past_due", repo.inv.Status)
	}

	// Redelivery: the same return event must not reopen or reverse twice — the
	// invoice is already past_due, so ReverseToUnpaid is never re-invoked.
	if err := h.handleACHReturn(context.Background(), refundEvt("pi_ret", stripe.RefundStatusSucceeded)); err != nil {
		t.Fatalf("handleACHReturn (redelivery): %v", err)
	}
	if repo.reverseCalls != 1 {
		t.Errorf("redelivered return re-reopened the invoice: %d ReverseToUnpaid calls, want 1", repo.reverseCalls)
	}
}

// A refund with no credit note and no tracked attempt (e.g. an unexpected card
// refund) is swallowed — nothing to attribute, and the webhook must still 200.
func TestACHWebhook_ReturnWithNoAttemptIsSwallowed(t *testing.T) {
	attempts := newMemAttempts()
	repo := &returnStubRepo{inv: &domain.Invoice{ID: uuid.New(), Status: domain.InvoiceStatusPaid}}
	subSvc := service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := &WebhookHandler{logger: slog.Default(), paymentAttempts: attempts, subService: subSvc}

	if err := h.handleACHReturn(context.Background(), refundEvt("pi_unknown", stripe.RefundStatusSucceeded)); err != nil {
		t.Fatalf("expected no error for an unattributable refund, got %v", err)
	}
	if repo.reverseCalls != 0 {
		t.Errorf("no invoice must be reopened when no attempt matches, got %d calls", repo.reverseCalls)
	}
}

// --- QA finding B: refund-vs-return classification guards ---

// classificationFixture seeds a settled us_bank_account attempt (amount 10000)
// on a paid invoice and returns the handler + spies.
func classificationFixture() (*WebhookHandler, *memAttempts, *returnStubRepo) {
	tenantID, invoiceID := uuid.New(), uuid.New()
	attempts := newMemAttempts()
	attempts.byPI["pi_cls"] = &domain.PaymentAttempt{
		GatewayPaymentIntentID: "pi_cls", Status: domain.PaymentAttemptSucceeded,
		Method: "us_bank_account", InvoiceID: invoiceID, TenantID: tenantID, Amount: 10000,
	}
	repo := &returnStubRepo{inv: &domain.Invoice{
		ID: invoiceID, TenantID: tenantID, Status: domain.InvoiceStatusPaid, Total: 10000,
	}}
	subSvc := service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return &WebhookHandler{logger: slog.Default(), paymentAttempts: attempts, subService: subSvc}, attempts, repo
}

// A merchant-initiated refund (dashboard refund carrying a merchant-set reason)
// must NOT be treated as a bank return: the invoice stays paid and the customer
// is never re-dunned for money the merchant gave back.
func TestACHWebhook_MerchantReasonRefundIsNotAReturn(t *testing.T) {
	h, attempts, repo := classificationFixture()
	ref := refundEvt("pi_cls", stripe.RefundStatusSucceeded)
	ref.Amount = 10000
	ref.Reason = stripe.RefundReasonRequestedByCustomer

	if err := h.handleACHReturn(context.Background(), ref); err != nil {
		t.Fatalf("handleACHReturn: %v", err)
	}
	if attempts.byPI["pi_cls"].Status != domain.PaymentAttemptSucceeded {
		t.Error("merchant refund must not mark the attempt returned")
	}
	if repo.reverseCalls != 0 || repo.inv.Status != domain.InvoiceStatusPaid {
		t.Errorf("merchant refund must not reopen the invoice: calls=%d status=%s", repo.reverseCalls, repo.inv.Status)
	}
}

// A PARTIAL refund can only be merchant-initiated — genuine ACH returns are
// always for the full debited amount. Must not reopen.
func TestACHWebhook_PartialRefundIsNotAReturn(t *testing.T) {
	h, attempts, repo := classificationFixture()
	ref := refundEvt("pi_cls", stripe.RefundStatusSucceeded)
	ref.Amount = 2500 // partial

	if err := h.handleACHReturn(context.Background(), ref); err != nil {
		t.Fatalf("handleACHReturn: %v", err)
	}
	if attempts.byPI["pi_cls"].Status != domain.PaymentAttemptSucceeded || repo.reverseCalls != 0 {
		t.Errorf("partial refund must not be treated as a return: status=%s calls=%d",
			attempts.byPI["pi_cls"].Status, repo.reverseCalls)
	}
}

// An attempt on a non-bank-debit method can never be a bank return.
func TestACHWebhook_NonBankMethodIsNotAReturn(t *testing.T) {
	h, attempts, repo := classificationFixture()
	attempts.byPI["pi_cls"].Method = "card"
	ref := refundEvt("pi_cls", stripe.RefundStatusSucceeded)
	ref.Amount = 10000

	if err := h.handleACHReturn(context.Background(), ref); err != nil {
		t.Fatalf("handleACHReturn: %v", err)
	}
	if attempts.byPI["pi_cls"].Status != domain.PaymentAttemptSucceeded || repo.reverseCalls != 0 {
		t.Error("a card attempt's refund must not be treated as a bank return")
	}
}

// The genuine article still processes: full amount, Stripe-initiated (no
// merchant reason), us_bank_account — attempt returned + invoice reopened.
func TestACHWebhook_GenuineFullReturnStillProcesses(t *testing.T) {
	h, attempts, repo := classificationFixture()
	ref := refundEvt("pi_cls", stripe.RefundStatusSucceeded)
	ref.Amount = 10000 // full, no Reason set

	if err := h.handleACHReturn(context.Background(), ref); err != nil {
		t.Fatalf("handleACHReturn: %v", err)
	}
	if attempts.byPI["pi_cls"].Status != domain.PaymentAttemptReturned {
		t.Errorf("attempt status = %s, want returned", attempts.byPI["pi_cls"].Status)
	}
	if repo.reverseCalls != 1 || repo.inv.Status != domain.InvoiceStatusPastDue {
		t.Errorf("genuine return must reopen: calls=%d status=%s", repo.reverseCalls, repo.inv.Status)
	}
}
