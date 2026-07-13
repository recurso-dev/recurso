package worker

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// failingGSP always errors, forcing RetryFailedEInvoice down its failure path.
type failingGSP struct{}

func (failingGSP) GenerateIRN(context.Context, *domain.Invoice) (*port.EInvoiceResponse, error) {
	return nil, errors.New("IRP unavailable")
}
func (failingGSP) GenerateIRNFull(context.Context, *port.EInvoiceRequest) (*port.EInvoiceResponse, error) {
	return nil, errors.New("IRP unavailable")
}
func (failingGSP) CancelIRN(context.Context, string, string) error {
	return errors.New("IRP unavailable")
}
func (failingGSP) GetIRNByDocDetails(context.Context, string, string, string) (*port.EInvoiceResponse, error) {
	return nil, errors.New("IRP unavailable")
}

// TestEInvoiceRetryWorker_CountAdvancesOnFailure_Postgres proves the ENG-179
// fix: when a retry FAILS, the invoice's e_invoice_retry_count advances (so
// maxEInvoiceRetries is eventually reached). Before the fix, the worker's stale
// full-row Update clobbered the service's increment and the count never moved.
func TestEInvoiceRetryWorker_CountAdvancesOnFailure_Postgres(t *testing.T) {
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
		tenantID, "EI2-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	// An e-invoice-ELIGIBLE customer (India + GSTIN + business), so the retry
	// actually reaches the (failing) GSP instead of no-op'ing as ineligible.
	customerID := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, email, ledger_account_id, country, gstin, tax_type, created_at)
		VALUES ($1,$2,$3,$4,'India','29ABCDE1234F1Z5','business',NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())

	// A FAILED e-invoice below the ceiling, due for retry, retry_count = 2.
	invID := uuid.New()
	exec(`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied,
			status, invoice_number, e_invoice_status, e_invoice_retry_count, e_invoice_next_retry_at, created_at, due_date)
		VALUES ($1,$2,$3,'INR',10000,10000,0,0,'open',$4,'FAILED',2, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		invID, tenantID, customerID, "INV-EI2-"+invID.String()[:8])

	invoiceRepo := db.NewInvoiceRepository(conn)
	einvoiceSvc := service.NewEInvoiceService(failingGSP{}, invoiceRepo, db.NewCustomerRepository(sqlxConn),
		db.NewIRPConfigRepository(conn), db.NewGSTConfigRepository(conn))
	w := NewEInvoiceRetryWorker(invoiceRepo, einvoiceSvc)

	w.processRetries(ctx)

	var count int
	var nextRetry sql.NullTime
	if err := conn.QueryRowContext(ctx,
		`SELECT e_invoice_retry_count, e_invoice_next_retry_at FROM invoices WHERE id = $1`, invID).
		Scan(&count, &nextRetry); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if count != 3 {
		t.Fatalf("e_invoice_retry_count = %d, want 3 (advanced after a failed retry; a stale-copy clobber would leave it at 2)", count)
	}
	if !nextRetry.Valid {
		t.Error("e_invoice_next_retry_at is NULL, want a scheduled backoff (below the retry ceiling)")
	}
}

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
