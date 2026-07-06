package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/tigerbeetle"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// --- Mock repo for reconciliation tests ---

type mockReconciliationRepo struct {
	nonDraft int
	paid     int
	scopeErr error

	invoiceRows  []db.InvoiceLedgerMismatch
	invoiceTotal int
	invoiceErr   error

	paymentRows  []db.InvoiceLedgerMismatch
	paymentTotal int
	paymentErr   error

	orphanRows  []db.OrphanLedgerTransaction
	orphanTotal int
	orphanErr   error

	accounts    []*domain.LedgerAccount
	accountsErr error

	txCount    int
	txCountErr error

	txSummaries []db.LedgerTransactionSummary
	txSummErr   error

	gotLimits []int
}

func (m *mockReconciliationRepo) CountReconciliationScope(ctx context.Context, tenantID uuid.UUID) (int, int, error) {
	if m.scopeErr != nil {
		return 0, 0, m.scopeErr
	}
	return m.nonDraft, m.paid, nil
}

func (m *mockReconciliationRepo) GetInvoiceLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error) {
	m.gotLimits = append(m.gotLimits, limit)
	if m.invoiceErr != nil {
		return nil, 0, m.invoiceErr
	}
	return m.invoiceRows, m.invoiceTotal, nil
}

func (m *mockReconciliationRepo) GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error) {
	m.gotLimits = append(m.gotLimits, limit)
	if m.paymentErr != nil {
		return nil, 0, m.paymentErr
	}
	return m.paymentRows, m.paymentTotal, nil
}

func (m *mockReconciliationRepo) GetOrphanLedgerTransactions(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.OrphanLedgerTransaction, int, error) {
	m.gotLimits = append(m.gotLimits, limit)
	if m.orphanErr != nil {
		return nil, 0, m.orphanErr
	}
	return m.orphanRows, m.orphanTotal, nil
}

func (m *mockReconciliationRepo) GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error) {
	if m.accountsErr != nil {
		return nil, m.accountsErr
	}
	return m.accounts, nil
}

func (m *mockReconciliationRepo) CountLedgerTransactionsByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	if m.txCountErr != nil {
		return 0, m.txCountErr
	}
	return m.txCount, nil
}

func (m *mockReconciliationRepo) GetLedgerTransactionSummaries(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.LedgerTransactionSummary, error) {
	if m.txSummErr != nil {
		return nil, m.txSummErr
	}
	return m.txSummaries, nil
}

// fakeTBReader is a test double for the narrow TigerBeetle view the
// reconciler consumes (TBTransferReader).
type fakeTBReader struct {
	connected bool
	transfers map[uuid.UUID][]tigerbeetle.TransferRecord // keyed by account ID
	err       error
	gotCalls  int
}

func (f *fakeTBReader) Connected() bool { return f.connected }

func (f *fakeTBReader) EnumerateAccountTransfers(ctx context.Context, accountID uuid.UUID, maxTransfers int) ([]tigerbeetle.TransferRecord, error) {
	f.gotCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.transfers[accountID], nil
}

// newTBService builds a reconciliation service whose TB view is the given
// fake. Same package, so the unexported field is assigned directly.
func newTBService(repo *mockReconciliationRepo, tb TBTransferReader) *ReconciliationService {
	svc := NewReconciliationService(repo, nil)
	svc.tb = tb
	return svc
}

// --- Tests ---

func TestReconciliationCleanBooks(t *testing.T) {
	repo := &mockReconciliationRepo{nonDraft: 42, paid: 30}
	svc := NewReconciliationService(repo, nil)

	tenantID := uuid.New()
	report, err := svc.Run(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", report.TenantID, tenantID)
	}
	if report.InvoicesChecked != 42 {
		t.Errorf("InvoicesChecked = %d, want 42", report.InvoicesChecked)
	}
	if report.PaidInvoicesChecked != 30 {
		t.Errorf("PaidInvoicesChecked = %d, want 30", report.PaidInvoicesChecked)
	}
	if report.TotalDiscrepancies != 0 {
		t.Errorf("TotalDiscrepancies = %d, want 0", report.TotalDiscrepancies)
	}
	if len(report.Discrepancies) != 0 {
		t.Errorf("Discrepancies = %d entries, want 0", len(report.Discrepancies))
	}
	if report.Truncated {
		t.Error("Truncated = true, want false")
	}
	if report.TBCompared {
		t.Error("TBCompared = true, want false (TB not connected)")
	}
	if report.TBSkipReason == "" {
		t.Error("TBSkipReason must explain why TB was not compared")
	}
	if report.StartedAt.IsZero() || report.FinishedAt.IsZero() {
		t.Error("StartedAt/FinishedAt must be set")
	}
	if report.FinishedAt.Before(report.StartedAt) {
		t.Error("FinishedAt must not precede StartedAt")
	}
}

func TestReconciliationMissingInvoiceTransaction(t *testing.T) {
	invoiceID := uuid.New()
	repo := &mockReconciliationRepo{
		nonDraft:     5,
		paid:         2,
		invoiceRows:  []db.InvoiceLedgerMismatch{{InvoiceID: invoiceID, Expected: 118000, Found: 0, TxCount: 0}},
		invoiceTotal: 1,
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalDiscrepancies != 1 {
		t.Fatalf("TotalDiscrepancies = %d, want 1", report.TotalDiscrepancies)
	}
	if len(report.Discrepancies) != 1 {
		t.Fatalf("expected 1 listed discrepancy, got %d", len(report.Discrepancies))
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyMissingInvoiceTx {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyMissingInvoiceTx)
	}
	if d.InvoiceID == nil || *d.InvoiceID != invoiceID {
		t.Errorf("InvoiceID = %v, want %v", d.InvoiceID, invoiceID)
	}
	if d.ExpectedAmount != 118000 || d.FoundAmount != 0 {
		t.Errorf("amounts = (%d, %d), want (118000, 0)", d.ExpectedAmount, d.FoundAmount)
	}
}

func TestReconciliationInvoiceAmountMismatch(t *testing.T) {
	invoiceID := uuid.New()
	repo := &mockReconciliationRepo{
		invoiceRows:  []db.InvoiceLedgerMismatch{{InvoiceID: invoiceID, Expected: 5000, Found: 4500, TxCount: 1}},
		invoiceTotal: 1,
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Discrepancies) != 1 {
		t.Fatalf("expected 1 discrepancy, got %d", len(report.Discrepancies))
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyInvoiceAmountMismatch {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyInvoiceAmountMismatch)
	}
	if d.ExpectedAmount != 5000 || d.FoundAmount != 4500 {
		t.Errorf("amounts = (%d, %d), want (5000, 4500)", d.ExpectedAmount, d.FoundAmount)
	}
}

func TestReconciliationMissingPaymentTransaction(t *testing.T) {
	invoiceID := uuid.New()
	repo := &mockReconciliationRepo{
		paid:         1,
		paymentRows:  []db.InvoiceLedgerMismatch{{InvoiceID: invoiceID, Expected: 7500, Found: 0, TxCount: 0}},
		paymentTotal: 1,
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Discrepancies) != 1 {
		t.Fatalf("expected 1 discrepancy, got %d", len(report.Discrepancies))
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyMissingPaymentTx {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyMissingPaymentTx)
	}
	if d.InvoiceID == nil || *d.InvoiceID != invoiceID {
		t.Errorf("InvoiceID = %v, want %v", d.InvoiceID, invoiceID)
	}
	if d.ExpectedAmount != 7500 {
		t.Errorf("ExpectedAmount = %d, want 7500 (amount_paid)", d.ExpectedAmount)
	}
}

func TestReconciliationPaymentAmountMismatch(t *testing.T) {
	repo := &mockReconciliationRepo{
		paymentRows:  []db.InvoiceLedgerMismatch{{InvoiceID: uuid.New(), Expected: 1000, Found: 900, TxCount: 2}},
		paymentTotal: 1,
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Discrepancies) != 1 || report.Discrepancies[0].Type != DiscrepancyPaymentAmountMismatch {
		t.Fatalf("expected 1 %s discrepancy, got %+v", DiscrepancyPaymentAmountMismatch, report.Discrepancies)
	}
}

func TestReconciliationOrphanedTransaction(t *testing.T) {
	txID := uuid.New()
	refID := uuid.New()
	repo := &mockReconciliationRepo{
		orphanRows:  []db.OrphanLedgerTransaction{{TransactionID: txID, Code: 3, Amount: 2500, ReferenceID: refID}},
		orphanTotal: 1,
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Discrepancies) != 1 {
		t.Fatalf("expected 1 discrepancy, got %d", len(report.Discrepancies))
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyOrphanedTransaction {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyOrphanedTransaction)
	}
	if d.TransactionID == nil || *d.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", d.TransactionID, txID)
	}
	if d.ReferenceID == nil || *d.ReferenceID != refID {
		t.Errorf("ReferenceID = %v, want %v", d.ReferenceID, refID)
	}
	if d.InvoiceID != nil {
		t.Errorf("InvoiceID = %v, want nil (reference matches no invoice)", d.InvoiceID)
	}
	if d.FoundAmount != 2500 {
		t.Errorf("FoundAmount = %d, want 2500", d.FoundAmount)
	}
}

func TestReconciliationCapsListedDiscrepancies(t *testing.T) {
	// Each repo query returns up to the cap; totals exceed what's listed.
	invoiceRows := make([]db.InvoiceLedgerMismatch, MaxListedDiscrepancies)
	for i := range invoiceRows {
		invoiceRows[i] = db.InvoiceLedgerMismatch{InvoiceID: uuid.New(), Expected: 100, Found: 0, TxCount: 0}
	}
	paymentRows := make([]db.InvoiceLedgerMismatch, 40)
	for i := range paymentRows {
		paymentRows[i] = db.InvoiceLedgerMismatch{InvoiceID: uuid.New(), Expected: 100, Found: 50, TxCount: 1}
	}
	repo := &mockReconciliationRepo{
		invoiceRows:  invoiceRows,
		invoiceTotal: 250, // huge drift; repo capped rows at limit
		paymentRows:  paymentRows,
		paymentTotal: 40,
		orphanTotal:  7, // counted but rows omitted by repo limit in this scenario
	}
	svc := NewReconciliationService(repo, nil)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Discrepancies) != MaxListedDiscrepancies {
		t.Errorf("listed discrepancies = %d, want cap %d", len(report.Discrepancies), MaxListedDiscrepancies)
	}
	if report.TotalDiscrepancies != 297 {
		t.Errorf("TotalDiscrepancies = %d, want 297 (250+40+7)", report.TotalDiscrepancies)
	}
	if !report.Truncated {
		t.Error("Truncated = false, want true")
	}
	// Repo queries must be limit-bounded, never unbounded.
	for i, limit := range repo.gotLimits {
		if limit != MaxListedDiscrepancies {
			t.Errorf("query %d called with limit %d, want %d", i, limit, MaxListedDiscrepancies)
		}
	}
}

func TestReconciliationTBClientNotConnected(t *testing.T) {
	// A zero-value LedgerClient has no live TB connection; the comparison
	// pass must skip honestly rather than pretend an empty ledger matched.
	repo := &mockReconciliationRepo{}
	svc := NewReconciliationService(repo, &tigerbeetle.LedgerClient{})

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TBCompared {
		t.Error("TBCompared = true, want false (client not connected)")
	}
	if report.TBSkipReason == "" {
		t.Error("TBSkipReason must explain the skipped TB comparison")
	}
}

func TestReconciliationTBMatchingSets(t *testing.T) {
	cashID, arID := uuid.New(), uuid.New()
	tx1, tx2, ref := uuid.New(), uuid.New(), uuid.New()

	repo := &mockReconciliationRepo{
		accounts: []*domain.LedgerAccount{{ID: cashID}, {ID: arID}},
		txCount:  2,
		txSummaries: []db.LedgerTransactionSummary{
			{TransactionID: tx1, Amount: 5000},
			{TransactionID: tx2, Amount: 7500},
		},
	}
	// The same transfer is visible from both accounts (debit and credit
	// side); the comparison must dedupe by ID.
	transfers := []tigerbeetle.TransferRecord{
		{ID: tx1, DebitAccountID: arID, CreditAccountID: cashID, Amount: 5000, Code: 1, ReferenceID: ref},
		{ID: tx2, DebitAccountID: cashID, CreditAccountID: arID, Amount: 7500, Code: 3, ReferenceID: ref},
	}
	tb := &fakeTBReader{connected: true, transfers: map[uuid.UUID][]tigerbeetle.TransferRecord{
		cashID: transfers,
		arID:   transfers,
	}}
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.TBCompared {
		t.Fatalf("TBCompared = false (reason %q), want true", report.TBSkipReason)
	}
	if report.TBSkipReason != "" {
		t.Errorf("TBSkipReason = %q, want empty", report.TBSkipReason)
	}
	if report.TotalDiscrepancies != 0 || len(report.Discrepancies) != 0 {
		t.Errorf("discrepancies = %d listed / %d total, want 0/0: %+v",
			len(report.Discrepancies), report.TotalDiscrepancies, report.Discrepancies)
	}
	if report.TBAccountsChecked != 2 {
		t.Errorf("TBAccountsChecked = %d, want 2", report.TBAccountsChecked)
	}
	if report.TBTransfersChecked != 2 {
		t.Errorf("TBTransfersChecked = %d, want 2 (deduped)", report.TBTransfersChecked)
	}
	if tb.gotCalls != 2 {
		t.Errorf("enumeration calls = %d, want 2 (one per account)", tb.gotCalls)
	}
}

func TestReconciliationTBMissingInTigerBeetle(t *testing.T) {
	accID := uuid.New()
	txID := uuid.New()
	repo := &mockReconciliationRepo{
		accounts:    []*domain.LedgerAccount{{ID: accID}},
		txCount:     1,
		txSummaries: []db.LedgerTransactionSummary{{TransactionID: txID, Amount: 4200}},
	}
	tb := &fakeTBReader{connected: true} // TB has nothing
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.TBCompared {
		t.Fatalf("TBCompared = false (reason %q), want true", report.TBSkipReason)
	}
	if report.TotalDiscrepancies != 1 || len(report.Discrepancies) != 1 {
		t.Fatalf("expected exactly 1 discrepancy, got %d listed / %d total",
			len(report.Discrepancies), report.TotalDiscrepancies)
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyMissingInTigerBeetle {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyMissingInTigerBeetle)
	}
	if d.TransactionID == nil || *d.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", d.TransactionID, txID)
	}
	if d.ExpectedAmount != 4200 || d.FoundAmount != 0 {
		t.Errorf("amounts = (%d, %d), want (4200, 0)", d.ExpectedAmount, d.FoundAmount)
	}
}

func TestReconciliationTBMissingInPostgres(t *testing.T) {
	accID := uuid.New()
	txID := uuid.New()
	refID := uuid.New()
	repo := &mockReconciliationRepo{
		accounts: []*domain.LedgerAccount{{ID: accID}},
	}
	tb := &fakeTBReader{connected: true, transfers: map[uuid.UUID][]tigerbeetle.TransferRecord{
		accID: {{ID: txID, Amount: 999, Code: 1, ReferenceID: refID}},
	}}
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.TBCompared {
		t.Fatalf("TBCompared = false (reason %q), want true", report.TBSkipReason)
	}
	if report.TotalDiscrepancies != 1 || len(report.Discrepancies) != 1 {
		t.Fatalf("expected exactly 1 discrepancy, got %d listed / %d total",
			len(report.Discrepancies), report.TotalDiscrepancies)
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyMissingInPostgres {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyMissingInPostgres)
	}
	if d.TransactionID == nil || *d.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", d.TransactionID, txID)
	}
	if d.ReferenceID == nil || *d.ReferenceID != refID {
		t.Errorf("ReferenceID = %v, want %v", d.ReferenceID, refID)
	}
	if d.FoundAmount != 999 || d.ExpectedAmount != 0 {
		t.Errorf("amounts = (%d, %d), want (0, 999)", d.ExpectedAmount, d.FoundAmount)
	}
}

func TestReconciliationTBAmountMismatch(t *testing.T) {
	accID := uuid.New()
	txID := uuid.New()
	repo := &mockReconciliationRepo{
		accounts:    []*domain.LedgerAccount{{ID: accID}},
		txCount:     1,
		txSummaries: []db.LedgerTransactionSummary{{TransactionID: txID, Amount: 5000}},
	}
	tb := &fakeTBReader{connected: true, transfers: map[uuid.UUID][]tigerbeetle.TransferRecord{
		accID: {{ID: txID, Amount: 4500, Code: 1}},
	}}
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.TBCompared {
		t.Fatalf("TBCompared = false (reason %q), want true", report.TBSkipReason)
	}
	if report.TotalDiscrepancies != 1 || len(report.Discrepancies) != 1 {
		t.Fatalf("expected exactly 1 discrepancy, got %d listed / %d total",
			len(report.Discrepancies), report.TotalDiscrepancies)
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyTBAmountMismatch {
		t.Errorf("Type = %q, want %q", d.Type, DiscrepancyTBAmountMismatch)
	}
	if d.TransactionID == nil || *d.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", d.TransactionID, txID)
	}
	if d.ExpectedAmount != 5000 || d.FoundAmount != 4500 {
		t.Errorf("amounts = (%d, %d), want (5000, 4500)", d.ExpectedAmount, d.FoundAmount)
	}
}

func TestReconciliationTBEnumerationErrorSkips(t *testing.T) {
	accID := uuid.New()
	invoiceID := uuid.New()
	repo := &mockReconciliationRepo{
		accounts:     []*domain.LedgerAccount{{ID: accID}},
		invoiceRows:  []db.InvoiceLedgerMismatch{{InvoiceID: invoiceID, Expected: 100, Found: 0, TxCount: 0}},
		invoiceTotal: 1,
	}
	tb := &fakeTBReader{connected: true, err: errors.New("tb cluster unreachable")}
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("TB enumeration errors must not fail the report, got: %v", err)
	}
	if report.TBCompared {
		t.Error("TBCompared = true, want false (enumeration failed)")
	}
	if !strings.Contains(report.TBSkipReason, "tb cluster unreachable") {
		t.Errorf("TBSkipReason = %q, want it to carry the enumeration error", report.TBSkipReason)
	}
	// The Postgres-side reconciliation must be unaffected.
	if report.TotalDiscrepancies != 1 || len(report.Discrepancies) != 1 {
		t.Errorf("PG discrepancies lost: %d listed / %d total, want 1/1",
			len(report.Discrepancies), report.TotalDiscrepancies)
	}
}

func TestReconciliationTBRowGuardSkips(t *testing.T) {
	repo := &mockReconciliationRepo{
		accounts: []*domain.LedgerAccount{{ID: uuid.New()}},
		txCount:  MaxTBComparedRows + 1,
	}
	tb := &fakeTBReader{connected: true}
	svc := newTBService(repo, tb)

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TBCompared {
		t.Error("TBCompared = true, want false (row-count memory guard)")
	}
	if report.TBSkipReason == "" || !strings.Contains(report.TBSkipReason, "guard") {
		t.Errorf("TBSkipReason = %q, want it to mention the comparison guard", report.TBSkipReason)
	}
	if tb.gotCalls != 0 {
		t.Errorf("enumeration calls = %d, want 0 (guard must trip before enumerating)", tb.gotCalls)
	}
}

func TestReconciliationRepoErrorsPropagate(t *testing.T) {
	cases := []struct {
		name string
		repo *mockReconciliationRepo
	}{
		{"scope error", &mockReconciliationRepo{scopeErr: errors.New("pg down")}},
		{"invoice query error", &mockReconciliationRepo{invoiceErr: errors.New("pg down")}},
		{"payment query error", &mockReconciliationRepo{paymentErr: errors.New("pg down")}},
		{"orphan query error", &mockReconciliationRepo{orphanErr: errors.New("pg down")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewReconciliationService(tc.repo, nil)
			if _, err := svc.Run(context.Background(), uuid.New()); err == nil {
				t.Error("Run must surface repo errors, got nil")
			}
		})
	}
}
