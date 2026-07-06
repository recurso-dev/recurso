package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// --- Mocks ---

// mockRecoveredPaymentRepo mimics the real repository's idempotency: inserts
// keyed by invoice_id, duplicates silently ignored (ON CONFLICT DO NOTHING).
type mockRecoveredPaymentRepo struct {
	records   map[uuid.UUID]*domain.RecoveredPayment
	insertErr error
	inserts   int // raw Insert calls, including ignored conflicts

	totals  *domain.RecoveryTotals
	monthly []domain.RecoveryMonthBucket
}

func newMockRecoveredPaymentRepo() *mockRecoveredPaymentRepo {
	return &mockRecoveredPaymentRepo{records: make(map[uuid.UUID]*domain.RecoveredPayment)}
}

func (m *mockRecoveredPaymentRepo) Insert(ctx context.Context, rec *domain.RecoveredPayment) error {
	m.inserts++
	if m.insertErr != nil {
		return m.insertErr
	}
	if _, exists := m.records[rec.InvoiceID]; exists {
		return nil // conflict ignored, first record wins
	}
	m.records[rec.InvoiceID] = rec
	return nil
}

func (m *mockRecoveredPaymentRepo) GetRecoveryTotals(ctx context.Context, tenantID uuid.UUID) (*domain.RecoveryTotals, error) {
	if m.totals != nil {
		return m.totals, nil
	}
	return &domain.RecoveryTotals{RecoveredAmountTotal: map[string]int64{}}, nil
}

func (m *mockRecoveredPaymentRepo) GetMonthlyRecoveries(ctx context.Context, tenantID uuid.UUID, months int) ([]domain.RecoveryMonthBucket, error) {
	return m.monthly, nil
}

type mockCampaignLookup struct {
	exec *domain.DunningCampaignExecution
	err  error
}

func (m *mockCampaignLookup) GetExecutionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.DunningCampaignExecution, error) {
	return m.exec, m.err
}

func recoveredTestInvoice(retryCount int) *domain.Invoice {
	now := time.Now().UTC()
	paidAt := now
	return &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Status:     domain.InvoiceStatusPaid,
		Total:      118000,
		Currency:   "INR",
		RetryCount: retryCount,
		CreatedAt:  now.Add(-5 * 24 * time.Hour),
		PaidAt:     &paidAt,
	}
}

// --- Qualification tests ---

func TestRecordIfRecovered_PaidWithRetries_Recorded(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "thompson_sampling")

	inv := recoveredTestInvoice(3)
	if !svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("expected invoice with retries to be recorded as recovered")
	}

	rec, ok := repo.records[inv.ID]
	if !ok {
		t.Fatal("expected a recovery record for the invoice")
	}
	if rec.TenantID != inv.TenantID {
		t.Errorf("TenantID = %v, want %v", rec.TenantID, inv.TenantID)
	}
	if rec.Amount != 118000 {
		t.Errorf("Amount = %d, want 118000", rec.Amount)
	}
	if rec.Currency != "INR" {
		t.Errorf("Currency = %q, want INR", rec.Currency)
	}
	if rec.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3 (retry_count at payment time)", rec.Attempts)
	}
	if rec.Strategy != "thompson_sampling" {
		t.Errorf("Strategy = %q, want thompson_sampling", rec.Strategy)
	}
	if rec.CampaignID != nil {
		t.Errorf("CampaignID = %v, want nil without a campaign execution", rec.CampaignID)
	}
	if rec.DaysToRecover != 5 {
		t.Errorf("DaysToRecover = %d, want 5", rec.DaysToRecover)
	}
	if !rec.RecoveredAt.Equal(inv.PaidAt.UTC()) {
		t.Errorf("RecoveredAt = %v, want PaidAt %v", rec.RecoveredAt, inv.PaidAt)
	}
}

func TestRecordIfRecovered_FirstTryPayment_NotRecorded(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "")

	inv := recoveredTestInvoice(0) // no retries, no dunning action
	if svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("first-try payment must not count as recovered")
	}
	if len(repo.records) != 0 {
		t.Errorf("expected 0 recovery records, got %d", len(repo.records))
	}
	if repo.inserts != 0 {
		t.Errorf("expected 0 insert calls, got %d", repo.inserts)
	}
}

func TestRecordIfRecovered_ActiveDunningAction_NoRetryCount_Recorded(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "")

	inv := recoveredTestInvoice(0)
	inv.DunningActionID = "24h" // scheduled by smart dunning before first retry ran
	if !svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("invoice with an active dunning action should qualify")
	}
	rec := repo.records[inv.ID]
	if rec == nil {
		t.Fatal("expected a recovery record")
	}
	if rec.Strategy != string(StrategyEpsilonGreedy) {
		t.Errorf("Strategy = %q, want default %q", rec.Strategy, StrategyEpsilonGreedy)
	}
}

func TestRecordIfRecovered_CampaignExecution_SetsCampaignAttribution(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "ucb1")
	campaignID := uuid.New()
	svc.SetCampaignLookup(&mockCampaignLookup{exec: &domain.DunningCampaignExecution{
		ID:         uuid.New(),
		CampaignID: campaignID,
	}})

	inv := recoveredTestInvoice(0) // campaign-managed invoice, no worker retries yet
	if !svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("campaign-managed invoice should qualify as recovered")
	}

	rec := repo.records[inv.ID]
	if rec == nil {
		t.Fatal("expected a recovery record")
	}
	if rec.Strategy != "campaign" {
		t.Errorf("Strategy = %q, want campaign", rec.Strategy)
	}
	if rec.CampaignID == nil || *rec.CampaignID != campaignID {
		t.Errorf("CampaignID = %v, want %v", rec.CampaignID, campaignID)
	}
}

func TestRecordIfRecovered_CampaignLookupError_StillRecordsOnRetries(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "")
	svc.SetCampaignLookup(&mockCampaignLookup{err: errors.New("db down")})

	inv := recoveredTestInvoice(2)
	if !svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("campaign lookup failure must not block recovery recording")
	}
	if repo.records[inv.ID] == nil {
		t.Fatal("expected a recovery record despite lookup error")
	}
}

func TestRecordIfRecovered_DoubleRecord_Idempotent(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	svc := NewDunningRecoveryService(repo, "")

	inv := recoveredTestInvoice(1)
	first := repo.inserts
	svc.RecordIfRecovered(context.Background(), inv)
	firstRec := repo.records[inv.ID]
	svc.RecordIfRecovered(context.Background(), inv) // e.g. webhook + worker race

	if repo.inserts != first+2 {
		t.Fatalf("expected 2 insert attempts, got %d", repo.inserts-first)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected exactly 1 stored record, got %d", len(repo.records))
	}
	if repo.records[inv.ID] != firstRec {
		t.Error("second record must not overwrite the first (conflict ignored)")
	}
}

func TestRecordIfRecovered_RepoError_NonFatal(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	repo.insertErr = errors.New("insert failed")
	svc := NewDunningRecoveryService(repo, "")

	inv := recoveredTestInvoice(2)
	if svc.RecordIfRecovered(context.Background(), inv) {
		t.Fatal("expected false when the insert fails")
	}
}

func TestRecordIfRecovered_NilInvoice_NoPanic(t *testing.T) {
	svc := NewDunningRecoveryService(newMockRecoveredPaymentRepo(), "")
	if svc.RecordIfRecovered(context.Background(), nil) {
		t.Fatal("nil invoice must not be recorded")
	}
}

// --- MarkInvoicePaid integration (payment-success hook) ---

func TestMarkInvoicePaid_WithRetries_RecordsRecovery(t *testing.T) {
	invoiceID := uuid.New()
	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:         invoiceID,
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Status:     domain.InvoiceStatusPastDue,
		Total:      50000,
		Currency:   "USD",
		RetryCount: 2,
		CreatedAt:  time.Now().Add(-48 * time.Hour),
	}}
	recRepo := newMockRecoveredPaymentRepo()
	recoverySvc := NewDunningRecoveryService(recRepo, "")

	svc := newMarkPaidService(invRepo, &mockLedgerRepoForMarkPaid{cashAccountID: uuid.New()})
	svc.SetRecoveryRecorder(recoverySvc)

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := recRepo.records[invoiceID]
	if rec == nil {
		t.Fatal("expected recovery record after marking a retried invoice paid")
	}
	if rec.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2", rec.Attempts)
	}
	if rec.Amount != 50000 || rec.Currency != "USD" {
		t.Errorf("Amount/Currency = %d/%s, want 50000/USD", rec.Amount, rec.Currency)
	}
}

func TestMarkInvoicePaid_FirstTry_NoRecovery(t *testing.T) {
	invoiceID := uuid.New()
	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:       invoiceID,
		Status:   domain.InvoiceStatusOpen,
		Total:    10000,
		Currency: "INR",
	}}
	recRepo := newMockRecoveredPaymentRepo()

	svc := newMarkPaidService(invRepo, &mockLedgerRepoForMarkPaid{cashAccountID: uuid.New()})
	svc.SetRecoveryRecorder(NewDunningRecoveryService(recRepo, ""))

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recRepo.records) != 0 {
		t.Errorf("first-try payment must not create a recovery record, got %d", len(recRepo.records))
	}
}

func TestMarkInvoicePaid_AlreadyPaid_NoDoubleRecovery(t *testing.T) {
	invoiceID := uuid.New()
	paidAt := time.Now().Add(-time.Hour)
	invRepo := &mockInvoiceRepoForMarkPaid{inv: &domain.Invoice{
		ID:         invoiceID,
		Status:     domain.InvoiceStatusPaid,
		PaidAt:     &paidAt,
		Total:      10000,
		RetryCount: 3,
	}}
	recRepo := newMockRecoveredPaymentRepo()

	svc := newMarkPaidService(invRepo, &mockLedgerRepoForMarkPaid{})
	svc.SetRecoveryRecorder(NewDunningRecoveryService(recRepo, ""))

	if err := svc.MarkInvoicePaid(context.Background(), invoiceID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoInserts := recRepo.inserts; repoInserts != 0 {
		t.Errorf("already-paid invoice must not trigger recovery recording, got %d inserts", repoInserts)
	}
}

// --- Summary shape ---

func TestGetRecoveredSummary_Shape(t *testing.T) {
	repo := newMockRecoveredPaymentRepo()
	repo.totals = &domain.RecoveryTotals{
		RecoveredAmountTotal: map[string]int64{"INR": 236000, "USD": 50000},
		RecoveredCount:       3,
		AvgAttempts:          2.5,
		AvgDaysToRecover:     4.0,
	}
	repo.monthly = []domain.RecoveryMonthBucket{
		{Month: "2026-06", Currency: "INR", Amount: 118000, Count: 1},
		{Month: "2026-07", Currency: "INR", Amount: 118000, Count: 1},
		{Month: "2026-07", Currency: "USD", Amount: 50000, Count: 1},
	}
	svc := NewDunningRecoveryService(repo, "")

	summary, err := svc.GetRecoveredSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.RecoveredCount != 3 {
		t.Errorf("RecoveredCount = %d, want 3", summary.RecoveredCount)
	}
	if summary.RecoveredAmountTotal["INR"] != 236000 {
		t.Errorf("INR total = %d, want 236000", summary.RecoveredAmountTotal["INR"])
	}
	if summary.AvgAttempts != 2.5 || summary.AvgDaysToRecover != 4.0 {
		t.Errorf("averages = %v/%v, want 2.5/4.0", summary.AvgAttempts, summary.AvgDaysToRecover)
	}
	if len(summary.Monthly) != 3 {
		t.Errorf("Monthly len = %d, want 3", len(summary.Monthly))
	}
}

func TestGetRecoveredSummary_EmptyIsNonNil(t *testing.T) {
	svc := NewDunningRecoveryService(newMockRecoveredPaymentRepo(), "")
	summary, err := svc.GetRecoveredSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.RecoveredAmountTotal == nil {
		t.Error("RecoveredAmountTotal must be an empty map, not nil")
	}
	if summary.Monthly == nil {
		t.Error("Monthly must be an empty slice, not nil")
	}
}
