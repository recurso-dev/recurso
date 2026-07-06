package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestRecoveredPaymentRepository_Postgres exercises the real SQL — insert
// idempotency (unique invoice_id), tenant-scoped totals, and the monthly
// series — against a throwaway postgres database. It applies the embedded
// migrations first, which also validates migration 000052.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database, e.g.:
//
//	createdb recurso_repo_test
//	TEST_DATABASE_URL='postgres://localhost:5432/recurso_repo_test?sslmode=disable' go test ./internal/adapter/db/
func TestRecoveredPaymentRepository_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed repository test")
	}

	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = conn.Close() }()

	repo := NewRecoveredPaymentRepository(conn)
	ctx := context.Background()

	tenantID := uuid.New()
	otherTenantID := uuid.New()
	now := time.Now().UTC()
	lastMonth := now.AddDate(0, -1, 0)
	campaignID := uuid.New()

	records := []*domain.RecoveredPayment{
		{ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(), Amount: 118000, Currency: "INR",
			Attempts: 2, Strategy: "epsilon_greedy", DaysToRecover: 3, RecoveredAt: now},
		{ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(), Amount: 50000, Currency: "USD",
			Attempts: 4, Strategy: "campaign", CampaignID: &campaignID, DaysToRecover: 7, RecoveredAt: lastMonth},
		{ID: uuid.New(), TenantID: otherTenantID, InvoiceID: uuid.New(), Amount: 99999, Currency: "EUR",
			Attempts: 1, Strategy: "ucb1", DaysToRecover: 1, RecoveredAt: now},
	}
	for _, rec := range records {
		if err := repo.Insert(ctx, rec); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	// Idempotency: re-inserting the same invoice must not error or duplicate.
	dup := *records[0]
	dup.ID = uuid.New()
	dup.Amount = 1 // must be ignored — first record wins
	if err := repo.Insert(ctx, &dup); err != nil {
		t.Fatalf("duplicate insert should be a silent no-op, got: %v", err)
	}

	totals, err := repo.GetRecoveryTotals(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetRecoveryTotals failed: %v", err)
	}
	if totals.RecoveredCount != 2 {
		t.Errorf("RecoveredCount = %d, want 2 (tenant-scoped, duplicate ignored)", totals.RecoveredCount)
	}
	if totals.RecoveredAmountTotal["INR"] != 118000 {
		t.Errorf("INR total = %d, want 118000", totals.RecoveredAmountTotal["INR"])
	}
	if totals.RecoveredAmountTotal["USD"] != 50000 {
		t.Errorf("USD total = %d, want 50000", totals.RecoveredAmountTotal["USD"])
	}
	if _, leaked := totals.RecoveredAmountTotal["EUR"]; leaked {
		t.Error("other tenant's currency leaked into totals")
	}
	if totals.AvgAttempts != 3.0 {
		t.Errorf("AvgAttempts = %v, want 3.0", totals.AvgAttempts)
	}
	if totals.AvgDaysToRecover != 5.0 {
		t.Errorf("AvgDaysToRecover = %v, want 5.0", totals.AvgDaysToRecover)
	}

	monthly, err := repo.GetMonthlyRecoveries(ctx, tenantID, 12)
	if err != nil {
		t.Fatalf("GetMonthlyRecoveries failed: %v", err)
	}
	if len(monthly) != 2 {
		t.Fatalf("monthly buckets = %d, want 2", len(monthly))
	}
	wantFirst := lastMonth.Format("2006-01")
	if monthly[0].Month != wantFirst || monthly[0].Currency != "USD" || monthly[0].Amount != 50000 || monthly[0].Count != 1 {
		t.Errorf("bucket[0] = %+v, want {%s USD 50000 1}", monthly[0], wantFirst)
	}
	wantSecond := now.Format("2006-01")
	if monthly[1].Month != wantSecond || monthly[1].Currency != "INR" || monthly[1].Amount != 118000 || monthly[1].Count != 1 {
		t.Errorf("bucket[1] = %+v, want {%s INR 118000 1}", monthly[1], wantSecond)
	}

	// A one-month window must exclude last month's recovery.
	windowed, err := repo.GetMonthlyRecoveries(ctx, tenantID, 1)
	if err != nil {
		t.Fatalf("GetMonthlyRecoveries(1) failed: %v", err)
	}
	if len(windowed) != 1 || windowed[0].Currency != "INR" {
		t.Errorf("1-month window = %+v, want only the current-month INR bucket", windowed)
	}
}
