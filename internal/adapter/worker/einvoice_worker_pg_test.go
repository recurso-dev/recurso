package worker

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/service"
)

// TestEInvoiceRetryWorker_MaxRetries_Postgres drives the e-invoice retry worker
// loop against a real DB: a FAILED e-invoice that has hit the retry ceiling is
// selected and stops being rescheduled (permanently failed). Guards the loop's
// selection + termination without needing a live GSP.
func TestEInvoiceRetryWorker_MaxRetries_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed einvoice worker test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	sqlxConn := sqlx.NewDb(conn, "postgres")
	ctx := context.Background()

	exec := func(q string, args ...any) {
		if _, err := conn.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}

	tenantID := uuid.New()
	exec(`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EI-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())

	// A FAILED e-invoice at the retry ceiling, due for retry.
	invID := uuid.New()
	exec(`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied,
			status, invoice_number, e_invoice_status, e_invoice_retry_count, e_invoice_next_retry_at, created_at, due_date)
		VALUES ($1,$2,$3,'INR',10000,10000,0,0,'open',$4,'FAILED',5, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		invID, tenantID, customerID, "INV-EI-"+invID.String()[:8])

	invoiceRepo := db.NewInvoiceRepository(conn)
	einvoiceSvc := service.NewEInvoiceService(nil, invoiceRepo, db.NewCustomerRepository(sqlxConn),
		db.NewIRPConfigRepository(conn), db.NewGSTConfigRepository(conn))
	w := NewEInvoiceRetryWorker(invoiceRepo, einvoiceSvc)

	w.processRetries(ctx)

	var nextRetry sql.NullTime
	var errMsg sql.NullString
	if err := conn.QueryRowContext(ctx,
		`SELECT e_invoice_next_retry_at, e_invoice_error_message FROM invoices WHERE id = $1`, invID).
		Scan(&nextRetry, &errMsg); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if nextRetry.Valid {
		t.Errorf("e_invoice_next_retry_at = %v, want NULL after the retry ceiling", nextRetry.Time)
	}
	if errMsg.String != "max retries exceeded" {
		t.Errorf("e_invoice_error_message = %q, want \"max retries exceeded\"", errMsg.String)
	}
}
