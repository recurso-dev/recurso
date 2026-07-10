package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestRevRecDeferral_Postgres proves the ENG-140 fix end-to-end against the real
// ledger schema: a subscription invoice credits Deferred Revenue (not Revenue),
// a one-off credits Revenue, recognition credits Recognized, and each recognition
// event posts once (attributable + idempotent under the ENG-142 unique index).
func TestRevRecDeferral_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed rev-rec test")
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
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "RevRec-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))
	subID := uuid.New()

	// Subscription invoice → credits DEFERRED (2100), never Revenue (4000).
	subInv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		SubscriptionID: &subID, InvoiceNumber: "SUB-1", Total: 120000, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, subInv); err != nil {
		t.Fatalf("RecordInvoice(subscription): %v", err)
	}
	if code := creditAccountCode(t, conn, subInv.ID, 1); code != domain.AccountCodeDeferredRevenue {
		t.Fatalf("subscription invoice credits account code %d, want %d (Deferred)", code, domain.AccountCodeDeferredRevenue)
	}

	// One-off invoice (no subscription) → credits REVENUE (4000).
	oneOff := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		InvoiceNumber: "ONE-1", Total: 5000, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, oneOff); err != nil {
		t.Fatalf("RecordInvoice(one-off): %v", err)
	}
	if code := creditAccountCode(t, conn, oneOff.ID, 1); code != domain.AccountCodeRevenue {
		t.Fatalf("one-off invoice credits account code %d, want %d (Revenue)", code, domain.AccountCodeRevenue)
	}

	// Recognition drains Deferred → Recognized, referenced by the event id.
	eventA := uuid.New()
	if _, err := svc.RecordRecognition(ctx, tenantID, 10000, eventA); err != nil {
		t.Fatalf("RecordRecognition: %v", err)
	}
	if code := creditAccountCode(t, conn, eventA, 2); code != domain.AccountCodeRecognizedRevenue {
		t.Fatalf("recognition credits account code %d, want %d (Recognized)", code, domain.AccountCodeRecognizedRevenue)
	}

	// Distinct events post distinct rows; a replayed event id posts once
	// (ENG-142 unique index compatibility — recognitions must not collide).
	eventB := uuid.New()
	if _, err := svc.RecordRecognition(ctx, tenantID, 10000, eventB); err != nil {
		t.Fatalf("RecordRecognition(B): %v", err)
	}
	if _, err := svc.RecordRecognition(ctx, tenantID, 10000, eventB); err != nil {
		t.Fatalf("RecordRecognition(B replay): %v", err)
	}
	if n := countTxByRef(t, conn, eventB, 2); n != 1 {
		t.Fatalf("replayed recognition event rows = %d, want 1 (idempotent)", n)
	}
	if n := countTxByRef(t, conn, eventA, 2); n != 1 {
		t.Fatalf("recognition event A rows = %d, want 1", n)
	}
}

func creditAccountCode(t *testing.T, conn *sql.DB, referenceID uuid.UUID, txCode int) int {
	t.Helper()
	var code int
	err := conn.QueryRowContext(context.Background(),
		`SELECT la.code FROM ledger_transactions lt
		 JOIN ledger_accounts la ON la.id = lt.credit_account_id
		 WHERE lt.reference_id = $1 AND lt.code = $2`, referenceID, txCode).Scan(&code)
	if err != nil {
		t.Fatalf("lookup credit account code (ref=%s, code=%d): %v", referenceID, txCode, err)
	}
	return code
}

func countTxByRef(t *testing.T, conn *sql.DB, referenceID uuid.UUID, txCode int) int {
	t.Helper()
	var n int
	if err := conn.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1 AND code = $2`, referenceID, txCode).Scan(&n); err != nil {
		t.Fatalf("count tx: %v", err)
	}
	return n
}
