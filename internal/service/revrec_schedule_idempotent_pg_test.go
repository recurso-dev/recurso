package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCreateScheduleForInvoice_Idempotent_Postgres proves a subscription
// invoice gets exactly ONE recognition schedule even when marked paid more than
// once (e.g. an ACH return reopens it and re-collection marks it paid again).
// Before the guard, a second full schedule was created — its events over-drained
// Deferred (negative) and double-recognized revenue.
func TestCreateScheduleForInvoice_Idempotent_Postgres(t *testing.T) {
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
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	rrSeed(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "RR-"+run, "rr-"+run+"@t.com")
	custID := uuid.New()
	rrSeed(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, "c-"+run+"@t.com", uuid.New())
	planID := uuid.New()
	rrSeed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'P','p-`+run+`','month',1,TRUE)`,
		planID, tenantID)

	subID := uuid.New()
	sub := &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: custID, PlanID: planID,
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().AddDate(0, 0, -1), CurrentPeriodEnd: time.Now().AddDate(0, 11, 0),
		BillingAnchor: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := db.NewSubscriptionRepository(conn).Create(ctx, sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	invID := uuid.New()
	inv := &domain.Invoice{
		ID: invID, TenantID: tenantID, CustomerID: custID, SubscriptionID: &subID,
		InvoiceNumber: "SUB-" + run, Status: domain.InvoiceStatusPaid,
		Currency: "USD", Total: 120000, TaxAmount: 0, CreatedAt: time.Now(), DueDate: time.Now(),
	}
	if err := db.NewInvoiceRepository(conn).Create(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	svc := NewRevRecService(db.NewRevRecRepository(conn), NewLedgerService(nil, db.NewLedgerRepository(conn)), nil)

	// Mark paid twice (settle → return → re-collect).
	if err := svc.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
		t.Fatalf("schedule 1: %v", err)
	}
	if err := svc.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
		t.Fatalf("schedule 2: %v", err)
	}

	var n int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM revenue_schedules WHERE invoice_id = $1 AND status = 'active'`, invID).Scan(&n); err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	if n != 1 {
		t.Errorf("active schedules for the invoice = %d, want 1 (a re-collected invoice must not create a second schedule → Deferred over-drain)", n)
	}
}

// exec is a small INSERT helper for the seed rows.
func rrSeed(t *testing.T, conn *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}
