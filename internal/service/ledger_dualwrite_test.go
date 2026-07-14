package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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

func (m *mockLedgerRepoForLedger) CreateTransactions(ctx context.Context, txs []*domain.LedgerTransaction) error {
	for _, tx := range txs {
		if err := m.CreateTransaction(ctx, tx); err != nil {
			return err
		}
	}
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
	taxAcctID := uuid.New()
	invoiceID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeRevenue:    {ID: revenueAcctID, Code: domain.AccountCodeRevenue},
		domain.AccountCodeTaxPayable: {ID: taxAcctID, Code: domain.AccountCodeTaxPayable},
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

	// ENG-159: Code-1 posts the gross to Revenue; a separate reclassification
	// (LedgerCodeOutputTax) moves the GST from Revenue to Tax Payable.
	if len(repo.transactions) != 2 {
		t.Fatalf("expected 2 transactions (invoice + GST reclass), got %d", len(repo.transactions))
	}
	var invoiceTx, taxTx *domain.LedgerTransaction
	for _, tx := range repo.transactions {
		if tx.ReferenceID != invoiceID {
			t.Errorf("ReferenceID = %v, want %v", tx.ReferenceID, invoiceID)
		}
		switch tx.Code {
		case 1:
			invoiceTx = tx
		case domain.LedgerCodeOutputTax:
			taxTx = tx
		}
	}
	if invoiceTx == nil || taxTx == nil {
		t.Fatalf("missing a leg: invoice=%v tax=%v", invoiceTx, taxTx)
	}
	// Code-1: debit AR, credit Revenue, GROSS total (the reconciler expects this).
	if invoiceTx.DebitAccountID != customerID || invoiceTx.CreditAccountID != revenueAcctID || invoiceTx.Amount != 118000 {
		t.Errorf("invoice leg = {debit %v credit %v amount %d}, want {AR %v, Revenue %v, 118000}",
			invoiceTx.DebitAccountID, invoiceTx.CreditAccountID, invoiceTx.Amount, customerID, revenueAcctID)
	}
	// Reclass: debit Revenue, credit Tax Payable, GST only → nets Revenue to subtotal.
	if taxTx.DebitAccountID != revenueAcctID || taxTx.CreditAccountID != taxAcctID || taxTx.Amount != 18000 {
		t.Errorf("GST reclass leg = {debit %v credit %v amount %d}, want {Revenue %v, Tax Payable %v, 18000}",
			taxTx.DebitAccountID, taxTx.CreditAccountID, taxTx.Amount, revenueAcctID, taxAcctID)
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

// TestLedgerRecordPayment_PostsCollectedNotGross proves the ENG-185 fix: when
// account credit was applied, the payment posts the CASH collected
// (Total - CreditApplied), not the gross Total — otherwise Cash is overstated
// and AR is over-credited (driven negative) by the applied credit.
func TestLedgerRecordPayment_PostsCollectedNotGross(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeCash: {ID: uuid.New(), Code: domain.AccountCodeCash},
	}}
	svc := NewLedgerService(nil, repo)

	// Total 1000, 300 covered by account credit -> only 700 cash collected.
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), InvoiceNumber: "INV-CR", Total: 1000, CreditApplied: 300}
	if err := svc.RecordPayment(context.Background(), inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(repo.transactions))
	}
	if got := repo.transactions[0].Amount; got != 700 {
		t.Fatalf("payment Amount = %d, want 700 (Total 1000 - credit 300)", got)
	}

	// Fully covered by credit -> no cash leg posted at all.
	repo.transactions = nil
	full := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), InvoiceNumber: "INV-FC", Total: 1000, CreditApplied: 1000}
	if err := svc.RecordPayment(context.Background(), full); err != nil {
		t.Fatalf("RecordPayment (full credit): %v", err)
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("fully-credit-covered invoice posted %d cash txns, want 0", len(repo.transactions))
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
	taxAcctID := uuid.New()
	subID := uuid.New()

	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{
		domain.AccountCodeDeferredRevenue: {ID: deferredAcctID, Code: domain.AccountCodeDeferredRevenue},
		domain.AccountCodeTaxPayable:      {ID: taxAcctID, Code: domain.AccountCodeTaxPayable},
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
	// ENG-159: Code-1 posts the gross to DEFERRED revenue; the reclassification
	// moves GST from Deferred to Tax Payable.
	if len(repo.transactions) != 2 {
		t.Fatalf("expected 2 transactions (deferred + GST reclass), got %d", len(repo.transactions))
	}
	var invoiceTx, taxTx *domain.LedgerTransaction
	for _, tx := range repo.transactions {
		switch tx.Code {
		case 1:
			invoiceTx = tx
		case domain.LedgerCodeOutputTax:
			taxTx = tx
		}
	}
	if invoiceTx == nil || taxTx == nil {
		t.Fatalf("missing a leg: invoice=%v tax=%v", invoiceTx, taxTx)
	}
	// Code-1: debit AR, credit DEFERRED revenue (not Revenue), gross total.
	if invoiceTx.DebitAccountID != customerID || invoiceTx.CreditAccountID != deferredAcctID || invoiceTx.Amount != 118000 {
		t.Errorf("invoice leg = {debit %v credit %v amount %d}, want {AR %v, Deferred %v, 118000}",
			invoiceTx.DebitAccountID, invoiceTx.CreditAccountID, invoiceTx.Amount, customerID, deferredAcctID)
	}
	// Reclass: debit Deferred, credit Tax Payable, GST only.
	if taxTx.DebitAccountID != deferredAcctID || taxTx.CreditAccountID != taxAcctID || taxTx.Amount != 18000 {
		t.Errorf("GST reclass leg = {debit %v credit %v amount %d}, want {Deferred %v, Tax Payable %v, 18000}",
			taxTx.DebitAccountID, taxTx.CreditAccountID, taxTx.Amount, deferredAcctID, taxAcctID)
	}
}
