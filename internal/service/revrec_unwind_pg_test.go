package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// seedSubAndInvoice inserts the customer/plan/subscription/invoice rows the
// revenue_schedules FKs require, and returns (subscriptionID, invoiceID,
// customerID). The invoice is a paid subscription invoice of `total`.
func seedSubAndInvoice(t *testing.T, conn *sql.DB, tenantID uuid.UUID, total int64) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	run := uuid.New().String()[:8]
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW(), NOW() + INTERVAL '1 month', NOW(), NOW(), NOW())`,
		subID, tenantID, customerID, planID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at)
		 VALUES ($1, $2, $3, $4, 'USD', $5, $5, $5, 'paid', $6, NOW())`,
		invID, tenantID, customerID, subID, total, "INV-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	return subID, invID, customerID
}

// seedRevRecSchedule inserts an active schedule for (tenant, invoice, sub) plus
// `months` pending recognition events of `perMonth`, dated one month apart.
func seedRevRecSchedule(t *testing.T, conn *sql.DB, tenantID, invoiceID, subID uuid.UUID, perMonth int64, months int) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	schedID := uuid.New()
	total := perMonth * int64(months)
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'USD', NOW(), NOW() + INTERVAL '1 month' * $6, 'active', NOW(), NOW())`,
		schedID, tenantID, invoiceID, subID, total, months); err != nil {
		t.Fatalf("seed schedule: %v", err)
	}
	for i := 0; i < months; i++ {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			 VALUES ($1,$2,$3,$4, NOW() + INTERVAL '1 month' * $5, 'pending', NOW())`,
			uuid.New(), schedID, tenantID, perMonth, i+1); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
	return schedID
}

func acctBalance(t *testing.T, conn *sql.DB, tenantID uuid.UUID, code int) int64 {
	t.Helper()
	var bal int64
	err := conn.QueryRowContext(context.Background(),
		`SELECT balance FROM ledger_accounts WHERE tenant_id = $1 AND code = $2`, tenantID, code).Scan(&bal)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		t.Fatalf("read balance code %d: %v", code, err)
	}
	return bal
}

func countEventsByStatus(t *testing.T, conn *sql.DB, schedID uuid.UUID, status string) int {
	t.Helper()
	var n int
	if err := conn.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM recognition_events WHERE revenue_schedule_id = $1 AND status = $2`, schedID, status).Scan(&n); err != nil {
		t.Fatalf("count events (%s): %v", status, err)
	}
	return n
}

func openRevRecTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed rev-rec unwind test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return conn
}

func seedRevRecTenant(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "RevRecUnwind-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenantID
}

// TestUnwindOnCancel_Postgres proves ENG-147 cancel: a mid-period immediate
// cancel voids all future recognition events and forfeits the remaining deferred
// as breakage revenue (DR Deferred / CR Recognized), so Deferred is drained and
// the worker can't keep recognizing.
func TestUnwindOnCancel_Postgres(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	subID, invID, customerID := seedSubAndInvoice(t, conn, tenantID, 120000)

	// Simulate the invoice's deferral: DR AR / CR Deferred 120000, then recognize
	// 2 of 12 months (20000) so 100000 remains deferred across 10 pending events.
	inv := &domain.Invoice{ID: invID, TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: "SUB-CANCEL", Total: 120000, Currency: "USD"}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	if _, err := ledger.RecordRecognition(ctx, tenantID, nil, 20000, uuid.New()); err != nil {
		t.Fatalf("seed recognition: %v", err)
	}
	schedID := seedRevRecSchedule(t, conn, tenantID, invID, subID, 10000, 10) // 100000 pending

	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 100000 {
		t.Fatalf("pre-cancel Deferred balance = %d, want 100000", b)
	}

	forfeited, err := svc.UnwindOnCancel(ctx, tenantID, subID)
	if err != nil {
		t.Fatalf("UnwindOnCancel: %v", err)
	}
	if forfeited != 100000 {
		t.Errorf("forfeited = %d, want 100000", forfeited)
	}
	// All pending events voided; none left pending.
	if n := countEventsByStatus(t, conn, schedID, "pending"); n != 0 {
		t.Errorf("pending events after cancel = %d, want 0", n)
	}
	if n := countEventsByStatus(t, conn, schedID, "canceled"); n != 10 {
		t.Errorf("canceled events = %d, want 10", n)
	}
	// Deferred drained to 0; the forfeited 100000 landed in Recognized (20000
	// earned + 100000 breakage = 120000).
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 0 {
		t.Errorf("post-cancel Deferred balance = %d, want 0", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 120000 {
		t.Errorf("Recognized balance = %d, want 120000", b)
	}
	// Schedule marked canceled.
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM revenue_schedules WHERE id = $1`, schedID).Scan(&status); err != nil {
		t.Fatalf("read schedule status: %v", err)
	}
	if status != domain.RevRecStatusCanceled {
		t.Errorf("schedule status = %q, want canceled", status)
	}

	// Idempotent: a second cancel finds no active schedule and forfeits nothing.
	if again, err := svc.UnwindOnCancel(ctx, tenantID, subID); err != nil || again != 0 {
		t.Errorf("second UnwindOnCancel = (%d, %v), want (0, nil)", again, err)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRecognizedRevenue); b != 120000 {
		t.Errorf("Recognized after idempotent re-run = %d, want 120000 (no double post)", b)
	}
}

// TestUnwindOnRefund_Postgres proves ENG-147 refund: a partial mid-period refund
// reverses only the unearned portion out of Deferred (DR Deferred / CR Refunds),
// reduces future events from the tail (splitting the boundary event), and leaves
// already-earned revenue untouched.
func TestUnwindOnRefund_Postgres(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	subID, invID, customerID := seedSubAndInvoice(t, conn, tenantID, 120000)

	// Deferral of 120000, none recognized yet: 12 pending events of 10000.
	inv := &domain.Invoice{ID: invID, TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: "SUB-REFUND", Total: 120000, Currency: "USD"}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	// The gateway cash refund that createRefund would post first (DR Refunds / CR Cash).
	creditNoteID := uuid.New()
	if err := ledger.RecordRefund(ctx, tenantID, nil, creditNoteID, 25000, "cash refund"); err != nil {
		t.Fatalf("RecordRefund: %v", err)
	}
	schedID := seedRevRecSchedule(t, conn, tenantID, invID, subID, 10000, 12) // 120000 pending

	// Refund 25000: crosses two 10000 events (voided) + reduces a third to 5000.
	reversed, err := svc.UnwindOnRefund(ctx, tenantID, nil, invID, creditNoteID, 25000)
	if err != nil {
		t.Fatalf("UnwindOnRefund: %v", err)
	}
	if reversed != 25000 {
		t.Errorf("reversed = %d, want 25000", reversed)
	}
	// Remaining pending deferred must be 120000 - 25000 = 95000.
	var remaining int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM recognition_events WHERE revenue_schedule_id = $1 AND status = 'pending'`,
		schedID).Scan(&remaining); err != nil {
		t.Fatalf("sum remaining pending: %v", err)
	}
	if remaining != 95000 {
		t.Errorf("remaining pending = %d, want 95000", remaining)
	}
	if n := countEventsByStatus(t, conn, schedID, "canceled"); n != 2 {
		t.Errorf("fully-voided events = %d, want 2", n)
	}
	// Ledger: Deferred = 120000 - 25000 reversed = 95000. Refunds expense
	// (25000 cash refund) fully offset by the 25000 deferred reversal → 0.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 95000 {
		t.Errorf("Deferred balance = %d, want 95000", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRefunds); b != 0 {
		t.Errorf("Refunds balance = %d, want 0 (unearned refund is not an expense)", b)
	}
	// The boundary event was split: exactly one pending event now carries 5000.
	var splitCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recognition_events WHERE revenue_schedule_id = $1 AND status = 'pending' AND amount = 5000`,
		schedID).Scan(&splitCount); err != nil {
		t.Fatalf("count split event: %v", err)
	}
	if splitCount != 1 {
		t.Errorf("split (5000) events = %d, want 1 (boundary event reduced)", splitCount)
	}
}

// TestUnwindOnRefund_FullRefund_Postgres proves a full mid-period refund (before
// any recognition) reverses all deferred and cancels the schedule.
func TestUnwindOnRefund_FullRefund_Postgres(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	subID, invID, customerID := seedSubAndInvoice(t, conn, tenantID, 60000)
	inv := &domain.Invoice{ID: invID, TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: "SUB-FULLREFUND", Total: 60000, Currency: "USD"}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	creditNoteID := uuid.New()
	if err := ledger.RecordRefund(ctx, tenantID, nil, creditNoteID, 60000, "full cash refund"); err != nil {
		t.Fatalf("RecordRefund: %v", err)
	}
	schedID := seedRevRecSchedule(t, conn, tenantID, invID, subID, 10000, 6)

	reversed, err := svc.UnwindOnRefund(ctx, tenantID, nil, invID, creditNoteID, 60000)
	if err != nil {
		t.Fatalf("UnwindOnRefund: %v", err)
	}
	if reversed != 60000 {
		t.Errorf("reversed = %d, want 60000", reversed)
	}
	if n := countEventsByStatus(t, conn, schedID, "pending"); n != 0 {
		t.Errorf("pending events after full refund = %d, want 0", n)
	}
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM revenue_schedules WHERE id = $1`, schedID).Scan(&status); err != nil {
		t.Fatalf("read schedule status: %v", err)
	}
	if status != domain.RevRecStatusCanceled {
		t.Errorf("schedule status = %q, want canceled (fully unwound)", status)
	}
	// Deferred fully drained; Refunds expense fully offset.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 0 {
		t.Errorf("Deferred balance = %d, want 0", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRefunds); b != 0 {
		t.Errorf("Refunds balance = %d, want 0", b)
	}
}

// TestUnwindOnRefund_TaxInvoice_NetUnwind_Postgres proves a PARTIAL refund of a
// TAX-bearing invoice unwinds the NET (ex-tax) amount from Deferred — the tax
// slice was already removed from Refunds by RecordRefundTaxReversal, and Deferred
// only ever held net revenue. This replicates the exact sequence credit_note.go
// runs. Passing the GROSS refund here (the pre-fix bug) reversed the tax twice:
// Refunds expense went to -5000 (abnormal credit balance) and Deferred was
// under-stated, breaking the deferred-revenue rollforward.
func TestUnwindOnRefund_TaxInvoice_NetUnwind_Postgres(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	svc := NewRevRecService(db.NewRevRecRepository(conn), ledger, nil)

	// Invoice total 110000 = net 100000 + tax 10000.
	subID, invID, customerID := seedSubAndInvoice(t, conn, tenantID, 110000)
	inv := &domain.Invoice{ID: invID, TenantID: tenantID, CustomerID: customerID,
		SubscriptionID: &subID, InvoiceNumber: "SUB-TAXREFUND", Total: 110000, TaxAmount: 10000, Currency: "USD"}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	// RecordInvoice defers NET: Deferred = 110000 - 10000 = 100000.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 100000 {
		t.Fatalf("pre-refund Deferred = %d, want 100000 (net)", b)
	}

	// The exact sequence credit_note.go runs on a partial refund of 55000:
	// gross cash refund, proportional tax reversal, then a NET deferred unwind.
	creditNoteID := uuid.New()
	refund := int64(55000)
	if err := ledger.RecordRefund(ctx, tenantID, nil, creditNoteID, refund, "cash refund"); err != nil {
		t.Fatalf("RecordRefund: %v", err)
	}
	tax := refundTaxPortion(refund, inv.TaxAmount, inv.Total) // round(55000*10000/110000) = 5000
	if tax != 5000 {
		t.Fatalf("refundTaxPortion = %d, want 5000", tax)
	}
	if _, err := ledger.RecordRefundTaxReversal(ctx, tenantID, nil, creditNoteID, tax, "gst reversal"); err != nil {
		t.Fatalf("RecordRefundTaxReversal: %v", err)
	}

	// Net deferred schedule: 100000 across 10 monthly events of 10000.
	seedRevRecSchedule(t, conn, tenantID, invID, subID, 10000, 10)

	// The fix: unwind NET (refund - tax) = 50000, not the gross 55000.
	netRefund := refund - tax
	reversed, err := svc.UnwindOnRefund(ctx, tenantID, nil, invID, creditNoteID, netRefund)
	if err != nil {
		t.Fatalf("UnwindOnRefund: %v", err)
	}
	if reversed != 50000 {
		t.Errorf("reversed = %d, want 50000 (net)", reversed)
	}
	// Refunds expense nets to 0: +55000 cash − 5000 tax reversal − 50000 net
	// deferred unwind. A gross (55000) unwind would leave Refunds at -5000.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeRefunds); b != 0 {
		t.Errorf("Refunds balance = %d, want 0 (gross unwind would give -5000 — the bug)", b)
	}
	// Deferred reduced by the net refund only: 100000 − 50000 = 50000.
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeDeferredRevenue); b != 50000 {
		t.Errorf("Deferred balance = %d, want 50000", b)
	}
}
