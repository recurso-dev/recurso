package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestClaimDueEvents_DisjointAndGuarded proves the F2 invariant: claiming due
// recognition events hands each event to exactly one worker, and the
// recognized/failed transitions only apply to events the caller claimed — so
// a losing worker's duplicate-post error can never demote an event the winner
// already recognized.
func TestClaimDueEvents_DisjointAndGuarded(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "F2-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())
	invID := uuid.New()
	mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',10000,10000,10000,0,'paid',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-F2-"+invID.String()[:8])
	schedID := uuid.New()
	mustExec(t, conn, `INSERT INTO revenue_schedules (id, tenant_id, invoice_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		VALUES ($1,$2,$3,10000,'USD', NOW() - INTERVAL '30 days', NOW() + INTERVAL '30 days', 'active', NOW(), NOW())`,
		schedID, tenantID, invID)
	eventID := uuid.New()
	mustExec(t, conn, `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
		VALUES ($1,$2,$3,5000, NOW() - INTERVAL '1 day', 'pending', NOW())`,
		eventID, schedID, tenantID)

	repo := db.NewRevRecRepository(conn)

	// Worker A claims the due event; worker B's claim must come back empty.
	first, err := repo.ClaimDueEvents(ctx, time.Now())
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	claimedMine := 0
	for _, e := range first {
		if e.ID == eventID {
			claimedMine++
		}
	}
	if claimedMine != 1 {
		t.Fatalf("first claim returned our event %d times, want 1", claimedMine)
	}
	second, err := repo.ClaimDueEvents(ctx, time.Now())
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	for _, e := range second {
		if e.ID == eventID {
			t.Fatalf("second claim also returned event %s — claims are not disjoint", eventID)
		}
	}

	// Winner recognizes; a late loser trying to mark it failed must be a no-op.
	if err := repo.MarkEventRecognized(ctx, eventID, uuid.New()); err != nil {
		t.Fatalf("MarkEventRecognized: %v", err)
	}
	if err := repo.MarkEventFailed(ctx, eventID, "duplicate posting"); err != nil {
		t.Fatalf("MarkEventFailed: %v", err)
	}
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM recognition_events WHERE id = $1`, eventID).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != domain.RecognitionStatusRecognized {
		t.Fatalf("status = %q after late failure mark, want %q (guard broken — F2)", status, domain.RecognitionStatusRecognized)
	}
}

// TestOneOffInvoice_ImmediateRecognition_NoDeferredDrain proves the F3
// invariant: a one-off (no-subscription) invoice recognizes NET revenue for
// reporting, pre-recognized, and the worker must post NOTHING to the ledger —
// one-off invoices credit Revenue directly at invoice time, so a Deferred
// drain here pushed the Deferred account negative by the invoice amount (the
// reconciler's abnormal_account_balance seen in production).
func TestOneOffInvoice_ImmediateRecognition_NoDeferredDrain(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "F3-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())
	invID := uuid.New()
	// GST invoice: total 11800 = 10000 net + 1800 tax. No subscription_id.
	mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, tax_amount, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'INR',10000,11800,1800,0,0,'paid',$4,NOW(),NOW())`,
		invID, tenantID, customerID, "INV-F3-"+invID.String()[:8])

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	if err := svc.CreateScheduleForInvoice(ctx, &domain.Invoice{
		ID: invID, TenantID: tenantID, CustomerID: customerID,
		Total: 11800, TaxAmount: 1800, Currency: "INR",
	}, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice: %v", err)
	}

	// Schedule carries NET revenue; its event is already recognized.
	var totalAmount int64
	var eventStatus string
	if err := conn.QueryRowContext(ctx,
		`SELECT s.total_amount, e.status FROM revenue_schedules s
		 JOIN recognition_events e ON e.revenue_schedule_id = s.id
		 WHERE s.invoice_id = $1`, invID).Scan(&totalAmount, &eventStatus); err != nil {
		t.Fatalf("read schedule+event: %v", err)
	}
	if totalAmount != 10000 {
		t.Errorf("schedule total = %d, want 10000 (net of tax — F3)", totalAmount)
	}
	if eventStatus != domain.RecognitionStatusRecognized {
		t.Errorf("event status = %q, want pre-recognized (no worker posting — F3)", eventStatus)
	}

	// The worker must have nothing to do, and the tenant must end with ZERO
	// ledger transactions: any posting here would be a Deferred drain that was
	// never funded.
	if err := svc.ProcessDueEvents(ctx); err != nil {
		t.Fatalf("ProcessDueEvents: %v", err)
	}
	var ledgerTx int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions t
		 JOIN ledger_accounts a ON a.id IN (t.debit_account_id, t.credit_account_id)
		 WHERE a.tenant_id = $1`, tenantID).Scan(&ledgerTx); err != nil {
		t.Fatalf("count ledger txs: %v", err)
	}
	if ledgerTx != 0 {
		t.Errorf("one-off immediate recognition posted %d ledger transaction(s), want 0 (Deferred drain — F3)", ledgerTx)
	}
}
