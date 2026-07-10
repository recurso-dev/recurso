package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mocks for ledger tests ---

type mockLedgerRepoForLedger struct {
	port.LedgerRepository
	accountsByCode  map[int]*domain.LedgerAccount
	lookupErr       error
	createTxErr     error
	transactions    []*domain.LedgerTransaction
	accountsCreated []*domain.LedgerAccount
}

func (m *mockLedgerRepoForLedger) GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error) {
	if m.lookupErr != nil {
		return nil, m.lookupErr
	}
	if acct, ok := m.accountsByCode[code]; ok {
		return acct, nil
	}
	return nil, errors.New("account not found")
}

func (m *mockLedgerRepoForLedger) CreateAccount(ctx context.Context, acc *domain.LedgerAccount) error {
	m.accountsCreated = append(m.accountsCreated, acc)
	return nil
}

func (m *mockLedgerRepoForLedger) CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error {
	if m.createTxErr != nil {
		return m.createTxErr
	}
	m.transactions = append(m.transactions, tx)
	return nil
}

// --- RecordInvoice tests ---

func TestLedgerRecordInvoice_DebitsARCreditsRevenue(t *testing.T) {
	customerID := uuid.New()
	revenueAcctID := uuid.New()
	invoiceID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeRevenue: {ID: revenueAcctID, Code: domain.AccountCodeRevenue},
	}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID:            invoiceID,
		TenantID:      uuid.New(),
		CustomerID:    customerID,
		InvoiceNumber: "INV-100",
		Subtotal:      100000,
		TaxAmount:     18000,
		Total:         118000,
	}

	if err := svc.RecordInvoice(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	tx := repo.transactions[0]
	// Amount must be the invoice total (subtotal + tax), not the subtotal.
	if tx.Amount != 118000 {
		t.Errorf("Amount = %d, want 118000 (invoice total)", tx.Amount)
	}
	if tx.DebitAccountID != customerID {
		t.Errorf("DebitAccountID = %v, want customer AR %v", tx.DebitAccountID, customerID)
	}
	if tx.CreditAccountID != revenueAcctID {
		t.Errorf("CreditAccountID = %v, want revenue account %v", tx.CreditAccountID, revenueAcctID)
	}
	if tx.ReferenceID != invoiceID {
		t.Errorf("ReferenceID = %v, want invoice %v", tx.ReferenceID, invoiceID)
	}
	if tx.Code != 1 {
		t.Errorf("Code = %d, want 1 (invoice)", tx.Code)
	}
}

func TestLedgerRecordInvoice_ProvisionsRevenueAccount(t *testing.T) {
	// A tenant without a chart of accounts gets one created on first
	// posting (the old hardcoded fallback UUIDs violated the FK).
	repo := &mockLedgerRepoForLedger{lookupErr: errors.New("no accounts")}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Total:      5000,
	}

	if err := svc.RecordInvoice(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	var revenueAcct *domain.LedgerAccount
	for _, a := range repo.accountsCreated {
		if a.Code == domain.AccountCodeRevenue {
			revenueAcct = a
		}
	}
	if revenueAcct == nil {
		t.Fatal("expected a Revenue account to be provisioned")
	}
	if repo.transactions[0].CreditAccountID != revenueAcct.ID {
		t.Errorf("CreditAccountID = %v, want provisioned revenue account %v", repo.transactions[0].CreditAccountID, revenueAcct.ID)
	}
}

func TestLedgerRecordInvoice_PGWriteFailureReturnsError(t *testing.T) {
	repo := &mockLedgerRepoForLedger{createTxErr: errors.New("pg down")}
	svc := NewLedgerService(nil, repo)

	err := svc.RecordInvoice(context.Background(), &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Total:      1000,
	})
	if err == nil {
		t.Fatal("RecordInvoice must surface a PG write failure, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Errorf("expected 0 persisted transactions, got %d", len(repo.transactions))
	}
}

func TestLedgerNegativeAmountsRejected(t *testing.T) {
	repo := &mockLedgerRepoForLedger{}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), Total: -500}
	if err := svc.RecordInvoice(context.Background(), inv); err == nil {
		t.Error("RecordInvoice must reject a negative total, got nil")
	}
	if err := svc.RecordPayment(context.Background(), inv); err == nil {
		t.Error("RecordPayment must reject a negative total, got nil")
	}
	if _, err := svc.RecordRecognition(context.Background(), inv.TenantID, -500, uuid.New()); err == nil {
		t.Error("RecordRecognition must reject a negative amount, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Errorf("expected 0 persisted transactions for negative amounts, got %d", len(repo.transactions))
	}
}

// --- RecordPayment tests ---

func TestLedgerRecordPayment_DebitsCashCreditsAR(t *testing.T) {
	customerID := uuid.New()
	cashAcctID := uuid.New()
	invoiceID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeCash: {ID: cashAcctID, Code: domain.AccountCodeCash},
	}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID:            invoiceID,
		TenantID:      uuid.New(),
		CustomerID:    customerID,
		InvoiceNumber: "INV-200",
		Total:         118000,
	}

	if err := svc.RecordPayment(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	tx := repo.transactions[0]
	if tx.Amount != 118000 {
		t.Errorf("Amount = %d, want 118000 (invoice total)", tx.Amount)
	}
	if tx.DebitAccountID != cashAcctID {
		t.Errorf("DebitAccountID = %v, want cash %v", tx.DebitAccountID, cashAcctID)
	}
	if tx.CreditAccountID != customerID {
		t.Errorf("CreditAccountID = %v, want customer AR %v", tx.CreditAccountID, customerID)
	}
	if tx.Code != 3 {
		t.Errorf("Code = %d, want 3 (payment)", tx.Code)
	}
	if tx.ReferenceID != invoiceID {
		t.Errorf("ReferenceID = %v, want %v", tx.ReferenceID, invoiceID)
	}
}

func TestLedgerRecordPayment_ProvisionsCashAccount(t *testing.T) {
	repo := &mockLedgerRepoForLedger{lookupErr: errors.New("no accounts")}
	svc := NewLedgerService(nil, repo)

	if err := svc.RecordPayment(context.Background(), &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Total:      7500,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	var cashAcct *domain.LedgerAccount
	for _, a := range repo.accountsCreated {
		if a.Code == domain.AccountCodeCash {
			cashAcct = a
		}
	}
	if cashAcct == nil {
		t.Fatal("expected a Cash account to be provisioned")
	}
	if repo.transactions[0].DebitAccountID != cashAcct.ID {
		t.Errorf("DebitAccountID = %v, want provisioned cash account %v", repo.transactions[0].DebitAccountID, cashAcct.ID)
	}
}

// --- RecordRecognition tests ---

func TestLedgerRecordRecognition_MovesDeferredToRecognized(t *testing.T) {
	deferredID := uuid.New()
	recognizedID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeDeferredRevenue:   {ID: deferredID, Code: domain.AccountCodeDeferredRevenue},
		domain.AccountCodeRecognizedRevenue: {ID: recognizedID, Code: domain.AccountCodeRecognizedRevenue},
	}}
	svc := NewLedgerService(nil, repo)

	eventID := uuid.New()
	txID, err := svc.RecordRecognition(context.Background(), uuid.New(), 4200, eventID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txID == uuid.Nil {
		t.Error("expected non-nil transaction ID")
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	tx := repo.transactions[0]
	if tx.Amount != 4200 {
		t.Errorf("Amount = %d, want 4200", tx.Amount)
	}
	if tx.DebitAccountID != deferredID {
		t.Errorf("DebitAccountID = %v, want deferred revenue %v", tx.DebitAccountID, deferredID)
	}
	if tx.CreditAccountID != recognizedID {
		t.Errorf("CreditAccountID = %v, want recognized revenue %v", tx.CreditAccountID, recognizedID)
	}
	if tx.Code != 2 {
		t.Errorf("Code = %d, want 2 (revenue recognition)", tx.Code)
	}
	if tx.ReferenceID != eventID {
		t.Errorf("ReferenceID = %v, want recognition event id %v (attributable)", tx.ReferenceID, eventID)
	}
}

// TestLedgerRecordInvoice_SubscriptionDefersRevenue proves the ENG-140 fix: a
// subscription invoice credits Deferred Revenue (not Revenue), so recognition
// can later drain Deferred → Recognized without double-booking.
func TestLedgerRecordInvoice_SubscriptionDefersRevenue(t *testing.T) {
	customerID := uuid.New()
	deferredAcctID := uuid.New()
	subID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeDeferredRevenue: {ID: deferredAcctID, Code: domain.AccountCodeDeferredRevenue},
	}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		CustomerID:     customerID,
		SubscriptionID: &subID, // subscription invoice → deferred
		InvoiceNumber:  "INV-SUB-1",
		Subtotal:       100000,
		TaxAmount:      18000,
		Total:          118000,
	}

	if err := svc.RecordInvoice(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	tx := repo.transactions[0]
	if tx.DebitAccountID != customerID {
		t.Errorf("DebitAccountID = %v, want customer AR %v", tx.DebitAccountID, customerID)
	}
	if tx.CreditAccountID != deferredAcctID {
		t.Errorf("CreditAccountID = %v, want DEFERRED revenue %v (not Revenue)", tx.CreditAccountID, deferredAcctID)
	}
	if tx.Amount != 118000 {
		t.Errorf("Amount = %d, want 118000 (invoice total)", tx.Amount)
	}
}
