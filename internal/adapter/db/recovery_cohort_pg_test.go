package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestRecoveryCohortWindow_Postgres proves the windowed recovery-rate inputs
// (QA finding D): both write-off paths stamp marked_uncollectible_at, and the
// two Since counts window correctly on their timestamps.
func TestRecoveryCohortWindow_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed recovery-cohort test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID, custID := seedCreditAppTenantCustomer(t, conn)
	invRepo := NewInvoiceRepository(conn).(*InvoiceRepository)
	recRepo := NewRecoveredPaymentRepository(conn)

	seedInv := func(status string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
			 VALUES ($1,$2,$3,'USD',1000,1000,0,0,$4,$5,NOW(),NOW() - INTERVAL '10 days')`,
			id, tenantID, custID, status, "RC-"+id.String()[:8]); err != nil {
			t.Fatalf("seed invoice: %v", err)
		}
		return id
	}

	// Both write-off paths stamp the timestamp.
	auto := seedInv("past_due")
	if err := invRepo.MarkAsUncollectible(ctx, auto); err != nil {
		t.Fatalf("MarkAsUncollectible: %v", err)
	}
	manual := seedInv("past_due")
	if ok, err := invRepo.MarkUncollectibleScoped(ctx, tenantID, manual); err != nil || !ok {
		t.Fatalf("MarkUncollectibleScoped: ok=%v err=%v", ok, err)
	}
	for _, id := range []uuid.UUID{auto, manual} {
		var ts *time.Time
		if err := conn.QueryRowContext(ctx,
			`SELECT marked_uncollectible_at FROM invoices WHERE id=$1`, id).Scan(&ts); err != nil {
			t.Fatalf("read stamp: %v", err)
		}
		if ts == nil {
			t.Fatalf("write-off of %s did not stamp marked_uncollectible_at", id)
		}
	}

	// An old write-off (backdated stamp) falls outside the window.
	old := seedInv("uncollectible")
	if _, err := conn.ExecContext(ctx,
		`UPDATE invoices SET marked_uncollectible_at = NOW() - INTERVAL '200 days' WHERE id=$1`, old); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	since := time.Now().AddDate(0, 0, -90)
	if n, err := invRepo.CountUncollectibleSince(ctx, tenantID, since); err != nil || n != 2 {
		t.Fatalf("CountUncollectibleSince = %d (err %v), want 2 (backdated excluded)", n, err)
	}

	// Recovered side: one recent, one ancient.
	insRec := func(at time.Time) {
		if err := recRepo.Insert(ctx, &domain.RecoveredPayment{
			ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(),
			Amount: 1000, Currency: "USD", Attempts: 2, Strategy: "epsilon_greedy",
			DaysToRecover: 3, RecoveredAt: at,
		}); err != nil {
			t.Fatalf("insert recovery: %v", err)
		}
	}
	insRec(time.Now().AddDate(0, 0, -5))
	insRec(time.Now().AddDate(0, 0, -180))
	_ = run

	if n, err := recRepo.CountRecoveredSince(ctx, tenantID, since); err != nil || n != 1 {
		t.Fatalf("CountRecoveredSince = %d (err %v), want 1 (ancient excluded)", n, err)
	}
}
