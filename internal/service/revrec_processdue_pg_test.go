package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// TestRevRecService_ProcessDueEvents_Postgres drives the revenue-recognition
// worker loop against a real DB: a due pending recognition event is recognized
// and posted to the ledger (DR Deferred Revenue / CR Recognized Revenue). This
// guards the revrec loop — a money/GAAP path — against a silent regression.
func TestRevRecService_ProcessDueEvents_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed revrec test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "RR-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())
	invID := uuid.New()
	mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',10000,10000,10000,0,'paid',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-RR-"+invID.String()[:8])
	schedID := uuid.New()
	mustExec(t, conn, `INSERT INTO revenue_schedules (id, tenant_id, invoice_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		VALUES ($1,$2,$3,10000,'USD', NOW() - INTERVAL '30 days', NOW() + INTERVAL '30 days', 'active', NOW(), NOW())`,
		schedID, tenantID, invID)
	eventID := uuid.New()
	mustExec(t, conn, `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
		VALUES ($1,$2,$3,5000, NOW() - INTERVAL '1 day', 'pending', NOW())`,
		eventID, schedID, tenantID)

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	if err := svc.ProcessDueEvents(ctx); err != nil {
		t.Fatalf("ProcessDueEvents: %v", err)
	}

	// The due event is recognized with a ledger transaction attached.
	var status string
	var ledgerTxID sql.NullString
	if err := conn.QueryRowContext(ctx,
		`SELECT status, ledger_tx_id FROM recognition_events WHERE id = $1`, eventID).
		Scan(&status, &ledgerTxID); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if status != "recognized" {
		t.Errorf("event status = %q, want recognized", status)
	}
	if !ledgerTxID.Valid {
		t.Error("ledger_tx_id should be set on a recognized event")
	}

	// A RevRec ledger transaction (code 2) posted for this event at the amount.
	var txCount int
	var amount int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(MAX(amount),0) FROM ledger_transactions WHERE reference_id = $1 AND code = 2`, eventID).
		Scan(&txCount, &amount); err != nil {
		t.Fatalf("read ledger tx: %v", err)
	}
	if txCount != 1 {
		t.Fatalf("revrec ledger transactions = %d, want 1", txCount)
	}
	if amount != 5000 {
		t.Errorf("revrec ledger amount = %d, want 5000", amount)
	}
}

func mustExec(t *testing.T, conn *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}
