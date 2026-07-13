package scheduler

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

type noopEmailSender struct{}

func (noopEmailSender) Send(_ context.Context, _ port.EmailMessage) error { return nil }

// TestDunningScheduler_ProcessDunning_Postgres drives the dunning scheduler's
// core loop against a real DB: an overdue open invoice is picked up, a retry is
// scheduled, and it is handed off to the smart-dunning worker. Regression guard
// for the recovery money path (ENG-162 test-coverage push).
func TestDunningScheduler_ProcessDunning_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed dunning e2e")
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
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Dun-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "Dunning Customer", "dun-"+customerID.String()[:8]+"@example.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// An open invoice that is past due and not yet dunning-managed.
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied,
			status, invoice_number, retry_count, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',10000,10000,0,0,'open',$4,0, NOW() - INTERVAL '10 days', NOW() - INTERVAL '5 days')`,
		invID, tenantID, customerID, "INV-DUN-"+invID.String()[:8]); err != nil {
		t.Fatalf("seed overdue invoice: %v", err)
	}

	invoiceRepo := db.NewInvoiceRepository(conn)
	notif := service.NewNotificationService(noopEmailSender{}, "http://example.test")
	sched := NewDunningScheduler(invoiceRepo, notif, memory.NewNoOpLocker(), DefaultDunningConfig(), "http://portal.test")

	sched.processDunning()

	// The overdue invoice should have a retry scheduled and be handed to the
	// smart-dunning worker.
	var retryCount int
	var managedBy sql.NullString
	var nextRetry sql.NullTime
	if err := conn.QueryRowContext(ctx,
		`SELECT retry_count, dunning_managed_by, next_retry_at FROM invoices WHERE id = $1`, invID).
		Scan(&retryCount, &managedBy, &nextRetry); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if retryCount != 1 {
		t.Errorf("retry_count = %d, want 1 (scheduler should have scheduled the first retry)", retryCount)
	}
	if managedBy.String != "worker" {
		t.Errorf("dunning_managed_by = %q, want \"worker\" (handed to smart dunning)", managedBy.String)
	}
	if !nextRetry.Valid {
		t.Error("next_retry_at should be set after dunning")
	}
}
