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

// --- Mocks for CreditNoteService tests ---

type cnRefundUpdate struct {
	id       uuid.UUID
	status   domain.CreditNoteRefundStatus
	refundID *string
	message  string
}

type mockCreditNoteRepo struct {
	created         []*domain.CreditNote
	createErr       error
	updates         []cnRefundUpdate
	updateErr       error
	refundedSum     int64
	sumErr          error
	existing        *domain.CreditNote // resolved by GetByRefundID / GetByID
	getByRefundErr  error
	getErr          error
	updateStatusErr error
	statusUpdates   int
}

// GetByID resolves a credit note from those created (or the seeded existing one).
func (m *mockCreditNoteRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.CreditNote, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, cn := range m.created {
		if cn.ID == id {
			return cn, nil
		}
	}
	if m.existing != nil && m.existing.ID == id {
		return m.existing, nil
	}
	return nil, nil
}

// UpdateStatus mirrors the real repo's conditional CAS: it only transitions when
// the current status matches oldStatus, and reports whether a row was updated.
func (m *mockCreditNoteRepo) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus domain.CreditNoteStatus, approverID uuid.UUID, approvedAt time.Time) (bool, error) {
	if m.updateStatusErr != nil {
		return false, m.updateStatusErr
	}
	for _, cn := range m.created {
		if cn.ID == id {
			if cn.Status != oldStatus {
				return false, nil // CAS miss — someone else moved it first
			}
			cn.Status = newStatus
			if approverID != uuid.Nil {
				cn.ApprovedBy = &approverID
				cn.ApprovedAt = &approvedAt
			}
			m.statusUpdates++
			return true, nil
		}
	}
	return false, nil
}

func (m *mockCreditNoteRepo) Create(ctx context.Context, cn *domain.CreditNote) error {
	if m.createErr != nil {
		return m.createErr
	}
	if cn.ID == uuid.Nil {
		cn.ID = uuid.New() // the real repo assigns the id via RETURNING
	}
	m.created = append(m.created, cn)
	return nil
}

func (m *mockCreditNoteRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.CreditNoteFilter) ([]*domain.CreditNote, error) {
	return m.created, nil
}

func (m *mockCreditNoteRepo) UpdateRefund(ctx context.Context, id uuid.UUID, status domain.CreditNoteRefundStatus, refundID *string, message string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updates = append(m.updates, cnRefundUpdate{id: id, status: status, refundID: refundID, message: message})
	return nil
}

func (m *mockCreditNoteRepo) SumActiveRefundsForInvoice(ctx context.Context, invoiceID uuid.UUID) (int64, error) {
	return m.refundedSum, m.sumErr
}

func (m *mockCreditNoteRepo) CreateRefundWithinLimit(ctx context.Context, cn *domain.CreditNote, invoiceID uuid.UUID, amountPaid int64) (bool, error) {
	if m.sumErr != nil {
		return false, m.sumErr
	}
	if m.createErr != nil {
		return false, m.createErr
	}
	// Mirror the real repo: over-refund → nothing inserted.
	if cn.Amount+m.refundedSum > amountPaid {
		return false, nil
	}
	if cn.ID == uuid.Nil {
		cn.ID = uuid.New()
	}
	m.created = append(m.created, cn)
	return true, nil
}

func (m *mockCreditNoteRepo) GetByRefundID(ctx context.Context, refundID string) (*domain.CreditNote, error) {
	if m.getByRefundErr != nil {
		return nil, m.getByRefundErr
	}
	if m.existing != nil && m.existing.RefundID != nil && *m.existing.RefundID == refundID {
		return m.existing, nil
	}
	return nil, nil // like the real repo: unknown refund id is (nil, nil)
}

func (m *mockCreditNoteRepo) SumApplicableAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, entityID *uuid.UUID, currency string) (int64, error) {
	return 0, nil
}

func (m *mockCreditNoteRepo) ListApplicationsByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.CreditApplicationLine, error) {
	return nil, nil
}

func (m *mockCreditNoteRepo) ClaimExpiredCredits(ctx context.Context, now time.Time, limit int) ([]domain.CreditExpiry, error) {
	return nil, nil
}

func (m *mockCreditNoteRepo) ApplyAdjustmentCredits(ctx context.Context, tenantID, customerID uuid.UUID, entityID *uuid.UUID, currency string, invoiceID uuid.UUID, invoiceTotal int64) (int64, error) {
	return 0, nil
}

type mockCNCustomerRepo struct {
	customer *domain.Customer
	err      error
}

func (m *mockCNCustomerRepo) List(context.Context, uuid.UUID, domain.CustomerFilter) ([]*domain.Customer, error) {
	if m.customer != nil {
		return []*domain.Customer{m.customer}, nil
	}
	return nil, m.err
}

func (m *mockCNCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.customer, nil
}

type mockCNInvoiceRepo struct {
	inv *domain.Invoice
	err error
}

func (m *mockCNInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.inv, nil
}

type cnRefundCall struct {
	paymentID string
	amount    int64
	currency  string
}

type mockCNGateway struct {
	port.PaymentGateway
	calls  []cnRefundCall
	result *port.RefundResult
	err    error
}

func (m *mockCNGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	m.calls = append(m.calls, cnRefundCall{paymentID: paymentID, amount: amount, currency: currency})
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &port.RefundResult{RefundID: "rfnd_test_1", Status: "processed"}, nil
}

// --- Fixtures ---

type cnFixture struct {
	tenantID   uuid.UUID
	customerID uuid.UUID
	invoiceID  uuid.UUID
	repo       *mockCreditNoteRepo
	gateway    *mockCNGateway
	invRepo    *mockCNInvoiceRepo
	svc        *CreditNoteService
}

func newCNFixture(inv *domain.Invoice) *cnFixture {
	f := &cnFixture{
		tenantID:   uuid.New(),
		customerID: uuid.New(),
		invoiceID:  uuid.New(),
		repo:       &mockCreditNoteRepo{},
		gateway:    &mockCNGateway{},
	}
	if inv != nil {
		inv.ID = f.invoiceID
		inv.TenantID = f.tenantID
		inv.CustomerID = f.customerID
	}
	f.invRepo = &mockCNInvoiceRepo{inv: inv}
	customerRepo := &mockCNCustomerRepo{customer: &domain.Customer{ID: f.customerID, TenantID: f.tenantID}}
	f.svc = NewCreditNoteService(f.repo, customerRepo, f.invRepo, f.gateway)
	return f
}

func paidInvoice(amountPaid int64, currency, gatewayPaymentID string) *domain.Invoice {
	return &domain.Invoice{
		InvoiceNumber:    "INV-100",
		Status:           domain.InvoiceStatusPaid,
		AmountPaid:       amountPaid,
		Total:            amountPaid,
		Currency:         currency,
		GatewayPaymentID: gatewayPaymentID,
	}
}

func refundRequest(f *cnFixture, amount int64, currency string) domain.CreateCreditNoteRequest {
	return domain.CreateCreditNoteRequest{
		CustomerID: f.customerID,
		InvoiceID:  &f.invoiceID,
		Amount:     amount,
		Currency:   currency,
		Reason:     "customer complaint",
		Type:       string(domain.CreditNoteTypeRefund),
	}
}

// --- Tests ---

// TestCreditNote_RejectsNonPositiveAmount proves the ENG-180 guard: a zero or
// negative credit note is refused for both refund and adjustment types, so a
// negative amount can never book negative account credit or call the gateway
// with a negative refund.
func TestCreditNote_RejectsNonPositiveAmount(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	for _, amt := range []int64{0, -1, -5000} {
		if _, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, amt, "INR")); err == nil {
			t.Errorf("refund amount %d: expected validation error, got nil", amt)
		}
		adj := domain.CreateCreditNoteRequest{CustomerID: f.customerID, Amount: amt, Currency: "INR", Type: string(domain.CreditNoteTypeAdjustment)}
		if _, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", adj); err == nil {
			t.Errorf("adjustment amount %d: expected validation error, got nil", amt)
		}
	}
	if len(f.gateway.calls) != 0 {
		t.Fatalf("gateway refund was called %d times for invalid amounts, want 0", len(f.gateway.calls))
	}
}

func TestCreditNote_Refund_CallsGatewayAndPersistsRefundID(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))

	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(f.gateway.calls) != 1 {
		t.Fatalf("expected 1 gateway refund call, got %d", len(f.gateway.calls))
	}
	call := f.gateway.calls[0]
	if call.paymentID != "pi_abc123" || call.amount != 400 || call.currency != "USD" {
		t.Errorf("gateway called with (%s, %d, %s); want (pi_abc123, 400, USD)", call.paymentID, call.amount, call.currency)
	}

	if cn.RefundStatus != domain.RefundStatusProcessed {
		t.Errorf("refund status = %s, want %s", cn.RefundStatus, domain.RefundStatusProcessed)
	}
	if cn.RefundID == nil || *cn.RefundID != "rfnd_test_1" {
		t.Errorf("refund id not persisted on credit note: %v", cn.RefundID)
	}
	if cn.Balance != 0 {
		t.Errorf("refund credit note balance = %d, want 0 (refunds are not spendable credit)", cn.Balance)
	}

	if len(f.repo.updates) != 1 {
		t.Fatalf("expected 1 UpdateRefund call, got %d", len(f.repo.updates))
	}
	up := f.repo.updates[0]
	if up.status != domain.RefundStatusProcessed || up.refundID == nil || *up.refundID != "rfnd_test_1" {
		t.Errorf("persisted refund state = (%s, %v), want (processed, rfnd_test_1)", up.status, up.refundID)
	}
}

func TestCreditNote_Refund_PendingGatewayStatusStaysPending(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "INR", "pay_xyz"))
	f.gateway.result = &port.RefundResult{RefundID: "rfnd_slow", Status: "pending"}

	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 250, "INR"))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if cn.RefundStatus != domain.RefundStatusPending {
		t.Errorf("refund status = %s, want pending", cn.RefundStatus)
	}
	if cn.RefundID == nil || *cn.RefundID != "rfnd_slow" {
		t.Errorf("refund id = %v, want rfnd_slow", cn.RefundID)
	}
}

func TestCreditNote_Refund_GatewayFailureMarksRefundFailed(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))
	f.gateway.err = errors.New("card network unavailable")

	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if cn.RefundStatus != domain.RefundStatusFailed {
		t.Fatalf("refund status = %s, want %s", cn.RefundStatus, domain.RefundStatusFailed)
	}
	if !strings.Contains(cn.RefundMessage, "card network unavailable") {
		t.Errorf("refund message %q should contain the gateway error", cn.RefundMessage)
	}
	if cn.RefundID != nil {
		t.Errorf("refund id should be nil on failure, got %v", *cn.RefundID)
	}
	if len(f.repo.updates) != 1 || f.repo.updates[0].status != domain.RefundStatusFailed {
		t.Errorf("refund_failed state was not persisted: %+v", f.repo.updates)
	}
}

func TestCreditNote_Refund_NoPaymentIDRequiresManualProcessing(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "")) // mock-era / offline payment

	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(f.gateway.calls) != 0 {
		t.Fatalf("gateway must not be called without a payment id, got %d calls", len(f.gateway.calls))
	}
	if cn.RefundStatus != domain.RefundStatusManualRequired {
		t.Fatalf("refund status = %s, want %s", cn.RefundStatus, domain.RefundStatusManualRequired)
	}
	if !strings.Contains(cn.RefundMessage, "no gateway payment id") {
		t.Errorf("refund message %q should explain why no refund was attempted", cn.RefundMessage)
	}
	if len(f.repo.created) != 1 {
		t.Errorf("credit note should still be created, got %d", len(f.repo.created))
	}
}

func TestCreditNote_Refund_OverRefundRejected(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 1200, "USD"))
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation, got %v", err)
	}
	if len(f.repo.created) != 0 {
		t.Errorf("no credit note should be created on over-refund, got %d", len(f.repo.created))
	}
	if len(f.gateway.calls) != 0 {
		t.Errorf("gateway must not be called on over-refund, got %d calls", len(f.gateway.calls))
	}
}

func TestCreditNote_Refund_CumulativeOverRefundRejected(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))
	f.repo.refundedSum = 700 // previously refunded via earlier credit notes

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation for cumulative over-refund, got %v", err)
	}

	// 300 remaining is still refundable
	if _, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 300, "USD")); err != nil {
		t.Fatalf("refund within remaining paid amount should succeed, got %v", err)
	}
}

func TestCreditNote_Refund_UnpaidInvoiceRejected(t *testing.T) {
	inv := paidInvoice(1000, "USD", "pi_abc123")
	inv.Status = domain.InvoiceStatusOpen
	f := newCNFixture(inv)

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation for unpaid invoice, got %v", err)
	}
	if len(f.gateway.calls) != 0 {
		t.Errorf("gateway must not be called for unpaid invoice")
	}
}

func TestCreditNote_Refund_RequiresInvoiceID(t *testing.T) {
	f := newCNFixture(nil)

	req := domain.CreateCreditNoteRequest{
		CustomerID: f.customerID,
		Amount:     100,
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeRefund),
	}
	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", req)
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation without invoice_id, got %v", err)
	}
}

func TestCreditNote_Refund_CurrencyMismatchRejected(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "INR"))
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation for currency mismatch, got %v", err)
	}
}

func TestCreditNote_Refund_WrongCustomerRejected(t *testing.T) {
	inv := paidInvoice(1000, "USD", "pi_abc123")
	f := newCNFixture(inv)
	inv.CustomerID = uuid.New() // invoice belongs to a different customer

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if !errors.Is(err, ErrCreditNoteValidation) {
		t.Fatalf("expected ErrCreditNoteValidation for wrong customer, got %v", err)
	}
}

func TestCreditNote_Refund_PersistFailureAfterGatewaySuccessSurfaces(t *testing.T) {
	f := newCNFixture(paidInvoice(1000, "USD", "pi_abc123"))
	f.repo.updateErr = errors.New("db down")

	_, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", refundRequest(f, 400, "USD"))
	if err == nil {
		t.Fatal("expected an error when the refund succeeded at the gateway but could not be persisted")
	}
	if !strings.Contains(err.Error(), "rfnd_test_1") {
		t.Errorf("error %q should mention the gateway refund id for reconciliation", err.Error())
	}
}

// --- Refund webhook consumption (ProcessGatewayRefundEvent) ---

// pendingRefundNote returns a fixture whose repo already holds a refund-type
// credit note with the given refund_status and gateway refund id.
func pendingRefundNote(status domain.CreditNoteRefundStatus, refundID string) (*cnFixture, *domain.CreditNote) {
	f := newCNFixture(nil)
	cn := &domain.CreditNote{
		ID:           uuid.New(),
		TenantID:     f.tenantID,
		CustomerID:   f.customerID,
		Type:         domain.CreditNoteTypeRefund,
		RefundStatus: status,
		RefundID:     &refundID,
	}
	f.repo.existing = cn
	return f, cn
}

func TestCreditNote_RefundWebhook_SuccessAdvancesPendingToProcessed(t *testing.T) {
	f, cn := pendingRefundNote(domain.RefundStatusPending, "rfnd_hook_1")

	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_1", true, ""); err != nil {
		t.Fatalf("ProcessGatewayRefundEvent returned error: %v", err)
	}

	if len(f.repo.updates) != 1 {
		t.Fatalf("expected 1 UpdateRefund call, got %d", len(f.repo.updates))
	}
	up := f.repo.updates[0]
	if up.id != cn.ID {
		t.Errorf("updated credit note %s, want %s", up.id, cn.ID)
	}
	if up.status != domain.RefundStatusProcessed {
		t.Errorf("status = %s, want %s", up.status, domain.RefundStatusProcessed)
	}
	if up.refundID == nil || *up.refundID != "rfnd_hook_1" {
		t.Errorf("refund id = %v, want rfnd_hook_1", up.refundID)
	}
}

func TestCreditNote_RefundWebhook_FailureRecordsGatewayReason(t *testing.T) {
	f, _ := pendingRefundNote(domain.RefundStatusPending, "re_hook_2")

	err := f.svc.ProcessGatewayRefundEvent(context.Background(), "re_hook_2", false, "expired_or_canceled_card")
	if err != nil {
		t.Fatalf("ProcessGatewayRefundEvent returned error: %v", err)
	}

	if len(f.repo.updates) != 1 {
		t.Fatalf("expected 1 UpdateRefund call, got %d", len(f.repo.updates))
	}
	up := f.repo.updates[0]
	if up.status != domain.RefundStatusFailed {
		t.Errorf("status = %s, want %s", up.status, domain.RefundStatusFailed)
	}
	if !strings.Contains(up.message, "expired_or_canceled_card") {
		t.Errorf("message %q should carry the gateway's failure reason", up.message)
	}
}

func TestCreditNote_RefundWebhook_FailureWithoutReasonStillExplains(t *testing.T) {
	f, _ := pendingRefundNote(domain.RefundStatusPending, "rfnd_hook_3")

	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_3", false, ""); err != nil {
		t.Fatalf("ProcessGatewayRefundEvent returned error: %v", err)
	}
	if len(f.repo.updates) != 1 || f.repo.updates[0].status != domain.RefundStatusFailed {
		t.Fatalf("refund_failed was not persisted: %+v", f.repo.updates)
	}
	if f.repo.updates[0].message == "" {
		t.Error("failure message must not be empty even when the gateway gives no reason")
	}
}

func TestCreditNote_RefundWebhook_AlreadyProcessedIsIdempotent(t *testing.T) {
	f, _ := pendingRefundNote(domain.RefundStatusProcessed, "rfnd_hook_4")

	// Redelivered success event: no error, no state change.
	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_4", true, ""); err != nil {
		t.Fatalf("redelivered success event should be a no-op, got %v", err)
	}
	// Late failure event after success was recorded: stored status stays.
	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_4", false, "bank rejected"); err != nil {
		t.Fatalf("late failure event should be a no-op, got %v", err)
	}

	if len(f.repo.updates) != 0 {
		t.Errorf("expected 0 UpdateRefund calls on non-pending credit note, got %d", len(f.repo.updates))
	}
}

func TestCreditNote_RefundWebhook_AlreadyFailedStaysFailed(t *testing.T) {
	f, _ := pendingRefundNote(domain.RefundStatusFailed, "rfnd_hook_5")

	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_5", true, ""); err != nil {
		t.Fatalf("event on refund_failed note should be a no-op, got %v", err)
	}
	if len(f.repo.updates) != 0 {
		t.Errorf("expected 0 UpdateRefund calls, got %d", len(f.repo.updates))
	}
}

func TestCreditNote_RefundWebhook_UnknownRefundIDTolerated(t *testing.T) {
	f := newCNFixture(nil) // repo holds no credit notes

	err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_stranger", true, "")
	if !errors.Is(err, ErrRefundNotFound) {
		t.Fatalf("expected ErrRefundNotFound for unknown refund id, got %v", err)
	}
	if len(f.repo.updates) != 0 {
		t.Errorf("expected 0 UpdateRefund calls for unknown refund id, got %d", len(f.repo.updates))
	}
}

func TestCreditNote_RefundWebhook_EmptyRefundIDRejected(t *testing.T) {
	f := newCNFixture(nil)

	if err := f.svc.ProcessGatewayRefundEvent(context.Background(), "", true, ""); !errors.Is(err, ErrRefundNotFound) {
		t.Fatalf("expected ErrRefundNotFound for empty refund id, got %v", err)
	}
}

func TestCreditNote_RefundWebhook_RepoErrorSurfaces(t *testing.T) {
	f, _ := pendingRefundNote(domain.RefundStatusPending, "rfnd_hook_6")
	f.repo.getByRefundErr = errors.New("db down")

	err := f.svc.ProcessGatewayRefundEvent(context.Background(), "rfnd_hook_6", true, "")
	if err == nil || errors.Is(err, ErrRefundNotFound) {
		t.Fatalf("repo errors must surface (so the gateway retries), got %v", err)
	}
}

func TestCreditNote_Adjustment_DoesNotTouchGateway(t *testing.T) {
	f := newCNFixture(nil)

	req := domain.CreateCreditNoteRequest{
		CustomerID: f.customerID,
		Amount:     500,
		Currency:   "USD",
		Reason:     "goodwill credit",
		// Type omitted — defaults to adjustment (pre-refund behavior)
	}
	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.Nil, "", req)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(f.gateway.calls) != 0 {
		t.Fatalf("adjustment credit notes must not call the gateway, got %d calls", len(f.gateway.calls))
	}
	if cn.Type != domain.CreditNoteTypeAdjustment {
		t.Errorf("type = %s, want adjustment", cn.Type)
	}
	if cn.RefundStatus != domain.RefundStatusNone {
		t.Errorf("refund status = %s, want none", cn.RefundStatus)
	}
	if cn.Balance != 500 {
		t.Errorf("adjustment balance = %d, want 500 (spendable credit)", cn.Balance)
	}
}

// --- Maker-checker (RBAC / segregation-of-duties) tests ---

const (
	roleSupport = "support"
	roleAdmin   = "admin"
)

// TestCreditNote_MakerChecker_SupportCreatesPending: a support user (maker) can
// only draft — the credit note lands in pending_approval and NO gateway refund
// fires until a checker approves.
func TestCreditNote_MakerChecker_SupportCreatesPending(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleSupport, refundRequest(f, 5000, "INR"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if cn.Status != domain.CreditNoteStatusPending {
		t.Errorf("support-created refund status = %q, want pending_approval", cn.Status)
	}
	if len(f.gateway.calls) != 0 {
		t.Errorf("pending credit note must not touch the gateway; got %d refund call(s)", len(f.gateway.calls))
	}
}

// A non-support caller (admin, or an API key with empty role) issues immediately.
func TestCreditNote_MakerChecker_AdminSelfIssues(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	cn, err := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleAdmin, refundRequest(f, 5000, "INR"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if cn.Status != domain.CreditNoteStatusIssued {
		t.Errorf("admin-created refund status = %q, want issued", cn.Status)
	}
	if len(f.gateway.calls) != 1 {
		t.Errorf("admin self-issue should refund immediately; got %d call(s)", len(f.gateway.calls))
	}
}

// A checker approving a pending refund issues it and executes exactly one refund.
func TestCreditNote_Approve_IssuesAndRefundsOnce(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	cn, _ := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleSupport, refundRequest(f, 5000, "INR"))

	approved, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, uuid.New())
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != domain.CreditNoteStatusIssued {
		t.Errorf("approved status = %q, want issued", approved.Status)
	}
	if len(f.gateway.calls) != 1 {
		t.Errorf("approval should fire exactly one refund; got %d", len(f.gateway.calls))
	}
}

// Double-approve is idempotent — the second call is a no-op and does NOT refund
// again (guards the check-then-act path).
func TestCreditNote_Approve_IdempotentNoDoubleRefund(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	cn, _ := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleSupport, refundRequest(f, 5000, "INR"))
	approver := uuid.New()

	if _, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, approver); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	if _, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, approver); err != nil {
		t.Fatalf("second approve: %v", err)
	}
	if len(f.gateway.calls) != 1 {
		t.Errorf("double approve must not double-refund; got %d gateway call(s)", len(f.gateway.calls))
	}
}

// The creator cannot approve their own credit note (segregation of duties), and
// no refund is executed on the blocked attempt.
func TestCreditNote_Approve_CreatorCannotSelfApprove(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	creator := uuid.New()
	cn, _ := f.svc.Create(context.Background(), f.tenantID, creator, roleSupport, refundRequest(f, 5000, "INR"))

	if _, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, creator); err == nil {
		t.Error("creator must not be able to approve their own credit note")
	}
	if len(f.gateway.calls) != 0 {
		t.Errorf("blocked self-approval must not refund; got %d call(s)", len(f.gateway.calls))
	}
	if cn.Status != domain.CreditNoteStatusPending {
		t.Errorf("status after blocked self-approval = %q, want still pending_approval", cn.Status)
	}
}

// Rejecting a pending credit note moves it to rejected without any refund.
func TestCreditNote_Reject_NoRefund(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	cn, _ := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleSupport, refundRequest(f, 5000, "INR"))

	rejected, err := f.svc.Reject(context.Background(), f.tenantID, cn.ID, uuid.New())
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != domain.CreditNoteStatusRejected {
		t.Errorf("rejected status = %q, want rejected", rejected.Status)
	}
	if len(f.gateway.calls) != 0 {
		t.Errorf("reject must not refund; got %d call(s)", len(f.gateway.calls))
	}
}

// TestCreditNote_Approve_NoRevertAfterGatewayCharged is the money-path guard for
// the revert edge case: if the gateway refund SUCCEEDS but persisting its result
// fails, Approve must NOT roll the credit note back to pending_approval — a
// re-approval would otherwise refund the customer a second time. The note stays
// issued and a retry fires no further gateway call.
func TestCreditNote_Approve_NoRevertAfterGatewayCharged(t *testing.T) {
	f := newCNFixture(paidInvoice(10000, "INR", "pay_123"))
	// Gateway refund succeeds, but persisting the refund result fails.
	f.repo.updateErr = errors.New("db write failed")
	cn, _ := f.svc.Create(context.Background(), f.tenantID, uuid.New(), roleSupport, refundRequest(f, 5000, "INR"))

	// First approval: gateway is charged once; post-processing fails but is swallowed.
	if _, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, uuid.New()); err != nil {
		t.Fatalf("approve should not surface an error once the gateway was charged: %v", err)
	}
	if cn.Status != domain.CreditNoteStatusIssued {
		t.Errorf("status = %q, want issued (must NOT revert to pending after a gateway charge)", cn.Status)
	}
	if len(f.gateway.calls) != 1 {
		t.Fatalf("expected exactly one gateway refund, got %d", len(f.gateway.calls))
	}

	// A re-approval must be a no-op — the note is already issued, so no second charge.
	if _, err := f.svc.Approve(context.Background(), f.tenantID, cn.ID, uuid.New()); err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	if len(f.gateway.calls) != 1 {
		t.Errorf("re-approval after a charged refund must NOT refund again; got %d gateway call(s)", len(f.gateway.calls))
	}
}
