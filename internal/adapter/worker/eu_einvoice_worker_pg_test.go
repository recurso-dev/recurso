package worker

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/einvoice_eu"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// euPGHarness opens the test DB, runs migrations, and returns a connection plus
// an exec helper. Shared setup for the postgres-backed EU worker tests.
func euPGHarness(t *testing.T) (*sql.DB, func(string, ...any)) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed EU e-invoice worker test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	exec := func(q string, args ...any) {
		if _, err := conn.ExecContext(context.Background(), q, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	return conn, exec
}

// seedEUInvoiceRow inserts a tenant, customer, invoice, and one eu_einvoices row
// with the given status/document/next_retry_at, returning the eu_einvoices id.
func seedEUInvoiceRow(t *testing.T, conn *sql.DB, exec func(string, ...any), status, document, nextRetrySQL string) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	exec(`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EU-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())
	invID := uuid.New()
	exec(`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied,
			status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'EUR',10000,10000,0,0,'open',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-EU-"+invID.String()[:8])
	euID := uuid.New()
	exec(`INSERT INTO eu_einvoices (id, tenant_id, invoice_id, syntax, status, document, recipient_vat_id,
			retry_count, next_retry_at, created_at, updated_at)
		VALUES ($1,$2,$3,'ubl21',$4,$5,'DE123456789',0, `+nextRetrySQL+`, NOW(), NOW())`,
		euID, tenantID, invID, status, document)
	return euID
}

func euRepoAndWorker(conn *sql.DB) (*db.EUInvoiceRepository, *EUEInvoiceRetryWorker) {
	repo := db.NewEUInvoiceRepository(conn)
	// Mock transport always succeeds, so a redrive delivers.
	svc := service.NewEUEInvoiceService(nil, repo, einvoice_eu.NewMockTransport())
	return repo, NewEUEInvoiceRetryWorker(repo, svc)
}

// A due, failed, document-bearing row is claimed, re-transmitted, and marked
// sent with its schedule cleared — the real SQL path (column order, lease,
// due-selection) end to end.
func TestEUEInvoiceRetryWorker_DeliversAndClears_Postgres(t *testing.T) {
	conn, exec := euPGHarness(t)
	euID := seedEUInvoiceRow(t, conn, exec, "failed", "<Invoice/>",
		`NOW() - INTERVAL '1 minute'`)

	_, w := euRepoAndWorker(conn)
	w.processRetries(context.Background())

	var status, msgID string
	var nextRetry sql.NullTime
	if err := conn.QueryRowContext(context.Background(),
		`SELECT status, message_id, next_retry_at FROM eu_einvoices WHERE id = $1`, euID).
		Scan(&status, &msgID, &nextRetry); err != nil {
		t.Fatalf("read eu_einvoice: %v", err)
	}
	if status != string(domain.EUInvoiceStatusSent) {
		t.Errorf("status = %q, want sent", status)
	}
	if msgID == "" {
		t.Error("message_id empty, want the transport id")
	}
	if nextRetry.Valid {
		t.Errorf("next_retry_at = %v, want NULL after delivery", nextRetry.Time)
	}
}

// The claim query must select ONLY due rows: a not-yet-due row, a row with no
// schedule, and a document-less generation failure are all left untouched.
func TestEUEInvoiceRetryWorker_ClaimSkipsIneligible_Postgres(t *testing.T) {
	conn, exec := euPGHarness(t)
	notDue := seedEUInvoiceRow(t, conn, exec, "failed", "<Invoice/>",
		`NOW() + INTERVAL '1 hour'`)
	noSchedule := seedEUInvoiceRow(t, conn, exec, "failed", "<Invoice/>", `NULL`)
	genFailure := seedEUInvoiceRow(t, conn, exec, "failed", "",
		`NOW() - INTERVAL '1 minute'`) // due but no document

	_, w := euRepoAndWorker(conn)
	w.processRetries(context.Background())

	for _, id := range []uuid.UUID{notDue, noSchedule, genFailure} {
		var status string
		if err := conn.QueryRowContext(context.Background(),
			`SELECT status FROM eu_einvoices WHERE id = $1`, id).Scan(&status); err != nil {
			t.Fatalf("read eu_einvoice %s: %v", id, err)
		}
		if status != string(domain.EUInvoiceStatusFailed) {
			t.Errorf("row %s status = %q, want it left 'failed' (ineligible for claim)", id, status)
		}
	}
}
