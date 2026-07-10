package db

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestMarkPaid_Concurrent_Postgres proves the ENG-142 race fix: when many
// settlers race to mark one invoice paid, the conditional UPDATE lets exactly
// one win, so side-effects (ledger post, recovered-revenue) run once.
func TestMarkPaid_Concurrent_Postgres(t *testing.T) {
	conn := openLedgerTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	now := time.Now().UTC()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Race-"+run, "race-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	invoiceID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, status, invoice_number, created_at)
		 VALUES ($1, $2, $3, 'INR', 100000, 118000, 'open', $4, NOW())`,
		invoiceID, tenantID, customerID, "INV-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	repo := NewInvoiceRepository(conn)
	const settlers = 12
	var wins int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < settlers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := repo.MarkPaid(ctx, invoiceID, now)
			if err == nil && ok {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("MarkPaid winners = %d, want exactly 1 (concurrent settlers must not all transition)", wins)
	}

	var status string
	var amountPaid int64
	if err := conn.QueryRowContext(ctx,
		`SELECT status, amount_paid FROM invoices WHERE id = $1`, invoiceID).Scan(&status, &amountPaid); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if status != "paid" || amountPaid != 118000 {
		t.Fatalf("invoice = (%s, %d), want (paid, 118000)", status, amountPaid)
	}
}

// TestLedgerCreateTransaction_Idempotent_Postgres proves the ledger backstop: a
// duplicate (reference_id, code) post is a no-op and moves balances once, while
// reference-less recognition rows (code 2) still post repeatedly.
func TestLedgerCreateTransaction_Idempotent_Postgres(t *testing.T) {
	conn := openLedgerTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	now := time.Now().UTC()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Ledger-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	debit := seedLedgerAccount(t, conn, tenantID, 101, "Cash", "asset")
	credit := seedLedgerAccount(t, conn, tenantID, 102, "AR", "asset")

	repo := NewLedgerRepository(conn)
	refID := uuid.New()
	const amt = 118000

	// Two posts of the same payment (same reference+code, different tx id).
	for i := 0; i < 2; i++ {
		if err := repo.CreateTransaction(ctx, &domain.LedgerTransaction{
			ID: uuid.New(), DebitAccountID: debit, CreditAccountID: credit,
			Amount: amt, LedgerID: 1, Code: 3, ReferenceID: refID, Timestamp: now,
		}); err != nil {
			t.Fatalf("CreateTransaction (payment %d): %v", i, err)
		}
	}

	if got := countTx(t, conn, refID, 3); got != 1 {
		t.Fatalf("payment rows for (ref, code=3) = %d, want 1 (idempotent)", got)
	}
	if b := balance(t, conn, debit); b != amt {
		t.Fatalf("debit balance = %d, want %d (applied once, not twice)", b, amt)
	}
	if b := balance(t, conn, credit); b != amt {
		t.Fatalf("credit balance = %d, want %d (applied once, not twice)", b, amt)
	}

	// Recognition rows carry the zero reference and legitimately post repeatedly.
	for i := 0; i < 2; i++ {
		if err := repo.CreateTransaction(ctx, &domain.LedgerTransaction{
			ID: uuid.New(), DebitAccountID: debit, CreditAccountID: credit,
			Amount: 5000, LedgerID: 1, Code: 2, Timestamp: now, // ReferenceID left zero
		}); err != nil {
			t.Fatalf("CreateTransaction (recognition %d): %v", i, err)
		}
	}
	// Scope the count to this test's freshly-created account so it is isolated
	// from recognition rows other tests leave in a reused scratch DB.
	var recCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE code = 2 AND debit_account_id = $1`, debit).Scan(&recCount); err != nil {
		t.Fatalf("count recognition rows: %v", err)
	}
	if recCount != 2 {
		t.Fatalf("recognition rows (zero ref, code=2) = %d, want 2 (not deduped)", recCount)
	}
}

// --- helpers ---

func openLedgerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed settlement test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return conn
}

func seedLedgerAccount(t *testing.T, conn *sql.DB, tenantID uuid.UUID, code int, name, typ string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO ledger_accounts (id, tenant_id, name, type, code, ledger_id, currency, balance)
		 VALUES ($1, $2, $3, $4, $5, 1, 'INR', 0)`,
		id, tenantID, name, typ, code); err != nil {
		t.Fatalf("seed ledger account: %v", err)
	}
	return id
}

func countTx(t *testing.T, conn *sql.DB, refID uuid.UUID, code int) int {
	t.Helper()
	var n int
	if err := conn.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1 AND code = $2`, refID, code).Scan(&n); err != nil {
		t.Fatalf("count tx: %v", err)
	}
	return n
}

func balance(t *testing.T, conn *sql.DB, accountID uuid.UUID) int64 {
	t.Helper()
	var b int64
	if err := conn.QueryRowContext(context.Background(),
		`SELECT balance FROM ledger_accounts WHERE id = $1`, accountID).Scan(&b); err != nil {
		t.Fatalf("read balance: %v", err)
	}
	return b
}
