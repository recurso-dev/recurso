package service

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestRevRecService_ProcessDueEvents_ConcurrentClaimsExclusive_Postgres proves
// the revenue-recognition worker is safe to run on multiple instances at once:
// two ProcessDueEvents loops racing over the SAME set of due events must
// recognize each event EXACTLY once — otherwise a duplicated DR Deferred / CR
// Recognized leg fabricates revenue on the P&L and over-drains Deferred (driving
// it negative). Also runs under `-race` to catch data races in the shared money
// path itself.
//
// Recognition is defended IN DEPTH, and this test locks in the composite
// guarantee:
//  1. the atomic claim — `UPDATE recognition_events SET status='processing'
//     WHERE status='pending' RETURNING` (ADR-003) — a losing worker's UPDATE
//     re-evaluates the WHERE after the winner commits and claims nothing; and
//  2. the ledger's idempotent insert on (reference_id=event id, code 2,
//     occurrence), so even if a second worker did post a recognition leg for the
//     same event it collides on the unique index and is a no-op.
//
// Verified: neutering layer 1 alone (dropping the `status='pending'` guard) does
// NOT double-post — layer 2 backstops it. A regression would have to defeat both.
func TestRevRecService_ProcessDueEvents_ConcurrentClaimsExclusive_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed revrec concurrency test")
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
		tenantID, "RRC-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())
	subID := uuid.New()
	planID := uuid.New()
	mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active, created_at) VALUES ($1,$2,'RRC Plan',$3,'month',1,true,NOW())`,
		planID, tenantID, "rrc-"+planID.String()[:8])
	mustExec(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW(), NOW() - INTERVAL '1 month', NOW(), NOW())`,
		subID, tenantID, customerID, planID)

	const n = 8
	const perEvent = int64(1000)
	total := int64(n) * perEvent

	// A subscription invoice funds Deferred = total, so correct recognition drains
	// it to exactly zero — a double-recognition would drive it negative.
	invID := uuid.New()
	invNo := "INV-RRC-" + invID.String()[:8]
	mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,$4,'USD',$5,$5,$5,'paid',$6,NOW(),NOW())`,
		invID, tenantID, customerID, subID, total, invNo)
	setupLedger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	if err := setupLedger.RecordInvoice(ctx, &domain.Invoice{
		ID: invID, TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: invNo, Total: total, Currency: "USD",
	}); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}

	schedID := uuid.New()
	mustExec(t, conn, `INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,'USD', NOW() - INTERVAL '30 days', NOW() + INTERVAL '30 days', 'active', NOW(), NOW())`,
		schedID, tenantID, invID, subID, total)
	eventIDs := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		eventIDs[i] = uuid.New()
		mustExec(t, conn, `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			VALUES ($1,$2,$3,$4, NOW() - INTERVAL '1 day', 'pending', NOW())`,
			eventIDs[i], schedID, tenantID, perEvent)
	}

	// Two independent workers (own service instances, shared DB) race the loop.
	var wg sync.WaitGroup
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := NewRevRecService(db.NewRevRecRepository(conn), NewLedgerService(nil, db.NewLedgerRepository(conn)), nil)
			if err := svc.ProcessDueEvents(ctx); err != nil {
				t.Errorf("ProcessDueEvents: %v", err)
			}
		}()
	}
	wg.Wait()

	// Every event recognized exactly once, none left pending/processing.
	var recognized, notRecognized int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FILTER (WHERE status='recognized'), COUNT(*) FILTER (WHERE status<>'recognized')
		 FROM recognition_events WHERE revenue_schedule_id = $1`, schedID).Scan(&recognized, &notRecognized); err != nil {
		t.Fatalf("count event statuses: %v", err)
	}
	if recognized != n || notRecognized != 0 {
		t.Fatalf("event statuses: recognized=%d not-recognized=%d, want %d / 0", recognized, notRecognized, n)
	}

	// Exactly ONE code-2 recognition leg per event — the crux: a double-claim
	// would post two legs for some event.
	var totalLegs, maxLegsForOneEvent int
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(c),0), COALESCE(MAX(c),0) FROM (
			SELECT reference_id, COUNT(*) AS c FROM ledger_transactions
			WHERE code = 2 AND reference_id = ANY($1) GROUP BY reference_id
		 ) s`, pqUUIDArray(eventIDs)).Scan(&totalLegs, &maxLegsForOneEvent); err != nil {
		t.Fatalf("count recognition legs: %v", err)
	}
	if totalLegs != n {
		t.Errorf("total recognition legs = %d, want %d (one per event)", totalLegs, n)
	}
	if maxLegsForOneEvent > 1 {
		t.Fatalf("an event was recognized %d times — the concurrent claim double-posted revenue", maxLegsForOneEvent)
	}

	// Deferred drained to exactly zero (total funded == total recognized).
	var deferredBalance int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(credits_posted - debits_posted),0) FROM ledger_accounts
		 WHERE tenant_id=$1 AND code=$2`, tenantID, domain.AccountCodeDeferredRevenue).Scan(&deferredBalance); err != nil {
		t.Fatalf("deferred balance: %v", err)
	}
	if deferredBalance != 0 {
		t.Errorf("Deferred balance = %d, want 0 (double-recognition would drive it negative)", deferredBalance)
	}
}

// pqUUIDArray renders a Postgres uuid[] literal for = ANY($1).
func pqUUIDArray(ids []uuid.UUID) string {
	out := "{"
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id.String()
	}
	return out + "}"
}
