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
