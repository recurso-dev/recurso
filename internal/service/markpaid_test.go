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

// --- Mocks for MarkInvoicePaid tests ---

type mockInvoiceRepoForMarkPaid struct {
	port.InvoiceRepository
	inv       *domain.Invoice
	getErr    error
	updateErr error
	updated   *domain.Invoice
}

func (m *mockInvoiceRepoForMarkPaid) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.inv, nil
}

func (m *mockInvoiceRepoForMarkPaid) Update(ctx context.Context, inv *domain.Invoice) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = inv
	return nil
}

// MarkPaid mirrors the real conditional UPDATE: it transitions an open invoice
// once and reports false if the invoice is missing or already paid.
func (m *mockInvoiceRepoForMarkPaid) MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) (bool, error) {
	if m.updateErr != nil {
		return false, m.updateErr
	}
	if m.inv == nil || m.inv.Status == domain.InvoiceStatusPaid {
		return false, nil
	}
	m.inv.Status = domain.InvoiceStatusPaid
	m.inv.PaidAt = &paidAt
	m.inv.AmountPaid = m.inv.Total
	m.updated = m.inv
	return true, nil
}

type mockLedgerRepoForMarkPaid struct {
	port.LedgerRepository
	cashAccountID uuid.UUID
	createTxErr   error
	transactions  []*domain.LedgerTransaction
}

func (m *mockLedgerRepoForMarkPaid) GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error) {
	if m.cashAccountID != uuid.Nil && code == domain.AccountCodeCash {
		return &domain.LedgerAccount{ID: m.cashAccountID, Code: code}, nil
	}
	return nil, errors.New("account not found")
}

func (m *mockLedgerRepoForMarkPaid) CreateAccount(ctx context.Context, acc *domain.LedgerAccount) error {
	return nil
}

func (m *mockLedgerRepoForMarkPaid) CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error {
	if m.createTxErr != nil {
		return m.createTxErr
	}
	m.transactions = append(m.transactions, tx)
	return nil
}

func newMarkPaidService(invRepo port.InvoiceRepository, ledgerRepo port.LedgerRepository) *SubscriptionService {
	return NewSubscriptionService(
		nil, // subRepo (unused: revrecService is nil)
		invRepo,
		nil, // planRepo
		nil, // customerRepo (unused: notificationService is nil)
		nil, // couponRepo
		&subMockNotifier{},
		NewLedgerService(nil, ledgerRepo),
		nil, // gateway
		nil, // gspAdapter
		nil, // txManager
		nil, // revrecService
		nil, // taxResolver
	)
}

// --- Tests ---

func TestMarkInvoicePaid_OpenToPaid(t *testing.T) {
	invoiceID := uuid.New()
	customerID := uuid.New()
	tenantID := uuid.New()
	cashAcctID := uuid.New()

	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:            invoiceID,
		TenantID:      tenantID,
		CustomerID:    customerID,
		InvoiceNumber: "INV-001",
		Status:        domain.InvoiceStatusOpen,
		Currency:      "INR",
		Subtotal:      100000,
		TaxAmount:     18000,
		Total:         118000,
	}}
	ledgerRepo := &mockLedgerRepoForMarkPaid{cashAccountID: cashAcctID}

	svc := newMarkPaidService(invRepo, ledgerRepo)

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if invRepo.updated == nil {
		t.Fatal("expected invoice to be updated")
	}
	if invRepo.updated.Status != domain.InvoiceStatusPaid {
		t.Errorf("Status = %q, want %q", invRepo.updated.Status, domain.InvoiceStatusPaid)
	}
	if invRepo.updated.PaidAt == nil {
		t.Error("expected PaidAt to be set")
	} else if time.Since(*invRepo.updated.PaidAt) > time.Minute {
		t.Errorf("PaidAt = %v, want ~now", *invRepo.updated.PaidAt)
	}
	if invRepo.updated.AmountPaid != 118000 {
		t.Errorf("AmountPaid = %d, want 118000 (invoice total)", invRepo.updated.AmountPaid)
	}

	// Ledger payment entry: Debit Cash, Credit Customer AR, amount = invoice total.
	if len(ledgerRepo.transactions) != 1 {
		t.Fatalf("expected 1 ledger transaction, got %d", len(ledgerRepo.transactions))
	}
	tx := ledgerRepo.transactions[0]
	if tx.Amount != 118000 {
		t.Errorf("ledger Amount = %d, want 118000 (invoice total)", tx.Amount)
	}
	if tx.DebitAccountID != cashAcctID {
		t.Errorf("DebitAccountID = %v, want cash account %v", tx.DebitAccountID, cashAcctID)
	}
	if tx.CreditAccountID != customerID {
		t.Errorf("CreditAccountID = %v, want customer AR %v", tx.CreditAccountID, customerID)
	}
	if tx.ReferenceID != invoiceID {
		t.Errorf("ReferenceID = %v, want invoice %v", tx.ReferenceID, invoiceID)
	}
	if tx.Code != 3 {
		t.Errorf("Code = %d, want 3 (payment)", tx.Code)
	}
}

func TestMarkInvoicePaid_AlreadyPaid_NoOp(t *testing.T) {
	invoiceID := uuid.New()
	paidAt := time.Now().Add(-24 * time.Hour)
	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:     invoiceID,
		Status: domain.InvoiceStatusPaid,
		PaidAt: &paidAt,
		Total:  5000,
	}}
	ledgerRepo := &mockLedgerRepoForMarkPaid{}

	svc := newMarkPaidService(invRepo, ledgerRepo)

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if invRepo.updated != nil {
		t.Error("already-paid invoice should not be updated again")
	}
	if len(ledgerRepo.transactions) != 0 {
		t.Errorf("expected 0 ledger transactions for already-paid invoice, got %d", len(ledgerRepo.transactions))
	}
}

func TestMarkInvoicePaid_NotFound(t *testing.T) {
	invRepo := &mockInvoiceRepoForMarkPaid{inv: nil} // GetByID returns nil, nil
	svc := newMarkPaidService(invRepo, &mockLedgerRepoForMarkPaid{})

	err := svc.MarkInvoicePaid(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for missing invoice")
	}
	if err.Error() != "invoice not found" {
		t.Errorf("error = %q, want 'invoice not found'", err.Error())
	}
}

func TestMarkInvoicePaid_GetError_Propagated(t *testing.T) {
	invRepo := &mockInvoiceRepoForMarkPaid{getErr: errors.New("db read failed")}
	svc := newMarkPaidService(invRepo, &mockLedgerRepoForMarkPaid{})

	if err := svc.MarkInvoicePaid(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error when invoice lookup fails")
	}
}

func TestMarkInvoicePaid_UpdateError_NoLedgerWrite(t *testing.T) {
	invoiceID := uuid.New()
	invRepo := &mockInvoiceRepoForMarkPaid{
		inv: &domain.Invoice{
			ID:     invoiceID,
			Status: domain.InvoiceStatusOpen,
			Total:  10000,
		},
		updateErr: errors.New("db write failed"),
	}
	ledgerRepo := &mockLedgerRepoForMarkPaid{}

	svc := newMarkPaidService(invRepo, ledgerRepo)

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err == nil {
		t.Fatal("expected error when invoice update fails")
	}
	// The ledger must not record a payment the DB never persisted.
	if len(ledgerRepo.transactions) != 0 {
		t.Errorf("expected 0 ledger transactions after failed update, got %d", len(ledgerRepo.transactions))
	}
}

func TestMarkInvoicePaid_LedgerWriteFails_InvoiceStillPaid(t *testing.T) {
	// Dual-write behavior: DB write succeeds, ledger (PG) write fails.
	// Current behavior: the ledger failure is swallowed (logged only) and
	// MarkInvoicePaid still succeeds — the invoice stays paid with no ledger entry.
	invoiceID := uuid.New()
	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:     invoiceID,
		Status: domain.InvoiceStatusOpen,
		Total:  25000,
	}}
	ledgerRepo := &mockLedgerRepoForMarkPaid{createTxErr: errors.New("ledger down")}

	svc := newMarkPaidService(invRepo, ledgerRepo)

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("MarkInvoicePaid should not fail on ledger error (current behavior), got: %v", err)
	}
	if invRepo.updated == nil || invRepo.updated.Status != domain.InvoiceStatusPaid {
		t.Error("invoice should still be marked paid despite ledger failure")
	}
	if len(ledgerRepo.transactions) != 0 {
		t.Errorf("expected 0 recorded ledger transactions, got %d", len(ledgerRepo.transactions))
	}
}
