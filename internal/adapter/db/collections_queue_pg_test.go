package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCollectionsQueue_Postgres proves the operator worklist (Collections
// Intelligence Inc 1): only invoices in a recovery state with a balance owing
// appear (paid/open excluded), the customer join + latest-attempt lateral work,
// days_overdue is computed, filters narrow, and count matches the list.
func TestCollectionsQueue_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed collections-queue test")
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
	repo := NewInvoiceRepository(conn)

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Coll-"+run, "coll-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, $5, NOW())`,
		custID, tenantID, "Acme "+run, "acme-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// seedInvoice inserts an invoice with an explicit status, due-date offset, and
	// paid amount so we can control amount_remaining (a generated column).
	seedInvoice := func(status string, total, paid int64, dueOffset string, managedBy string, lastErr string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, dunning_managed_by, last_payment_error, retry_count, created_at, due_date)
			 VALUES ($1, $2, $3, 'USD', $4, $4, $5, 0, $6, $7, $8, $9, 2, NOW(), NOW() - `+dueOffset+`)`,
			id, tenantID, custID, total, paid, status, "INV-"+id.String()[:8], managedBy, lastErr); err != nil {
			t.Fatalf("seed invoice (%s): %v", status, err)
		}
		return id
	}

	pastDueID := seedInvoice("past_due", 10000, 0, "INTERVAL '10 days'", "worker", "card_declined")
	uncollID := seedInvoice("uncollectible", 5000, 0, "INTERVAL '40 days'", "scheduler", "insufficient_funds")
	seedInvoice("paid", 3000, 3000, "INTERVAL '5 days'", "scheduler", "")  // excluded: paid / no balance
	seedInvoice("open", 2000, 0, "INTERVAL '1 day'", "scheduler", "")      // excluded: not a recovery status
	seedInvoice("past_due", 4000, 4000, "INTERVAL '3 days'", "worker", "") // excluded: balance fully covered

	// A settled-then-returned ACH attempt on the past_due invoice — the queue
	// should surface its latest attempt status.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO payment_attempts (id, tenant_id, invoice_id, gateway, method, gateway_payment_intent_id, status, amount, created_at)
		 VALUES ($1, $2, $3, 'stripe', 'us_bank_account', $4, 'returned', 10000, NOW())`,
		uuid.New(), tenantID, pastDueID, "pi_"+run); err != nil {
		t.Fatalf("seed payment attempt: %v", err)
	}

	// --- Unfiltered: exactly the two owing recovery invoices, oldest due first. ---
	all, err := repo.ListCollectionsQueue(ctx, tenantID, domain.CollectionsQueueFilter{Limit: 50})
	if err != nil {
		t.Fatalf("ListCollectionsQueue: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 queue items, got %d: %+v", len(all), all)
	}
	// uncollectible is 40 days overdue → older due date → sorts first.
	if all[0].ID != uncollID || all[1].ID != pastDueID {
		t.Errorf("wrong order: want [uncoll, pastdue], got [%s, %s]", all[0].ID, all[1].ID)
	}
	pd := all[1]
	if pd.CustomerName != "Acme "+run || pd.CustomerEmail != "acme-"+run+"@t.com" {
		t.Errorf("customer join wrong: %+v", pd)
	}
	if pd.AmountRemaining != 10000 {
		t.Errorf("amount_remaining = %d, want 10000", pd.AmountRemaining)
	}
	if pd.DaysOverdue < 9 || pd.DaysOverdue > 11 {
		t.Errorf("days_overdue = %d, want ~10", pd.DaysOverdue)
	}
	if pd.LastPaymentError != "card_declined" || pd.ManagedBy != "worker" {
		t.Errorf("recovery state wrong: err=%q managed_by=%q", pd.LastPaymentError, pd.ManagedBy)
	}
	if pd.AttemptStatus != "returned" {
		t.Errorf("attempt_status = %q, want returned", pd.AttemptStatus)
	}

	count, err := repo.CountCollectionsQueue(ctx, tenantID, domain.CollectionsQueueFilter{})
	if err != nil {
		t.Fatalf("CountCollectionsQueue: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// --- Filter by status. ---
	onlyUncoll, err := repo.ListCollectionsQueue(ctx, tenantID, domain.CollectionsQueueFilter{Status: "uncollectible", Limit: 50})
	if err != nil {
		t.Fatalf("ListCollectionsQueue(status): %v", err)
	}
	if len(onlyUncoll) != 1 || onlyUncoll[0].ID != uncollID {
		t.Errorf("status filter wrong: %+v", onlyUncoll)
	}

	// --- Filter by managed_by. ---
	onlyWorker, err := repo.ListCollectionsQueue(ctx, tenantID, domain.CollectionsQueueFilter{ManagedBy: "worker", Limit: 50})
	if err != nil {
		t.Fatalf("ListCollectionsQueue(managed_by): %v", err)
	}
	if len(onlyWorker) != 1 || onlyWorker[0].ID != pastDueID {
		t.Errorf("managed_by filter wrong: %+v", onlyWorker)
	}
	if n, _ := repo.CountCollectionsQueue(ctx, tenantID, domain.CollectionsQueueFilter{ManagedBy: "worker"}); n != 1 {
		t.Errorf("filtered count = %d, want 1", n)
	}
}
