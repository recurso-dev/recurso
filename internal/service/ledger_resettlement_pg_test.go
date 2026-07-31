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

// TestLedgerResettlementCycle_Postgres is the oracle for the occurrence-aware
// idempotency design (docs/design-ledger-occurrence.md), run against the REAL
// unique index and balance maintenance:
//
//	settle → return → RE-COLLECT → return again → collect again
//
// Before migration 000146 the re-collection's cash leg was silently swallowed
// by the (reference_id, code) dedup — invoice paid, Cash understated forever.
// This test proves: every cycle's legs post, same-cycle duplicates still dedup,
// the Cash ACCOUNT BALANCE nets to exactly one settlement, the reconciler stays
// clean on the healthy books, and the reconciler now FLAGS the pre-fix
// corruption shape (re-settle leg missing).
func TestLedgerResettlementCycle_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed resettlement-cycle test")
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

	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme', $4, NOW(), NOW())`,
		customerID, tenantID, "cyc-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))

	const total = int64(100000)
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
		InvoiceNumber: "CYC-" + run, Total: total, Currency: "USD",
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,NOW(),NOW())`,
		inv.ID, tenantID, customerID, total, inv.InvoiceNumber); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}

	legCount := func(code int) int {
		var n int
		if err := conn.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1 AND code=$2`,
			inv.ID, code).Scan(&n); err != nil {
			t.Fatalf("count code %d: %v", code, err)
		}
		return n
	}

	// --- Cycle 1: settle; a redelivered duplicate must dedup at the DB. ---
	if err := ledger.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 1: %v", err)
	}
	if err := ledger.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 1 (dup): %v", err)
	}
	if got := legCount(3); got != 1 {
		t.Fatalf("after settle + duplicate: %d cash legs, want 1", got)
	}

	// --- Bank return: reversal; duplicate reversal must dedup. ---
	if err := ledger.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 1: %v", err)
	}
	if err := ledger.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 1 (dup): %v", err)
	}
	if got := legCount(19); got != 1 {
		t.Fatalf("after reversal + duplicate: %d reversal legs, want 1", got)
	}

	// --- THE FIX: re-collection posts a fresh cash leg (occurrence 1). ---
	if err := ledger.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("re-settle: %v", err)
	}
	if got := legCount(3); got != 2 {
		t.Fatalf("after re-collection: %d cash legs, want 2 — the re-settle leg was swallowed", got)
	}
	var occ int
	if err := conn.QueryRowContext(ctx,
		`SELECT MAX(occurrence) FROM ledger_transactions WHERE reference_id=$1 AND code=3`,
		inv.ID).Scan(&occ); err != nil || occ != 1 {
		t.Fatalf("re-settle occurrence = %d (err %v), want 1", occ, err)
	}

	// --- Cycle 2: second return + third collection also land. ---
	if err := ledger.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 2: %v", err)
	}
	if err := ledger.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 3: %v", err)
	}
	if got, want := legCount(19), 2; got != want {
		t.Fatalf("reversal legs = %d, want %d", got, want)
	}
	if got, want := legCount(3), 3; got != want {
		t.Fatalf("cash legs = %d, want %d", got, want)
	}

	// --- The Cash ACCOUNT BALANCE (maintained by applyLedgerTx) nets to exactly
	// one settlement: 3 settles − 2 reversals = +total. ---
	var cashBalance int64
	if err := conn.QueryRowContext(ctx, `
		SELECT la.balance FROM ledger_accounts la
		WHERE la.id = (SELECT debit_account_id FROM ledger_transactions
		               WHERE reference_id=$1 AND code=3 LIMIT 1)`,
		inv.ID).Scan(&cashBalance); err != nil {
		t.Fatalf("read cash balance: %v", err)
	}
	if cashBalance != total {
		t.Fatalf("cash account balance = %d, want %d (one net settlement)", cashBalance, total)
	}

	// --- Reconciler: healthy post-cycle books raise NO payment discrepancy. ---
	recon := NewReconciliationService(db.NewLedgerRepository(conn), nil)
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("reconciliation Run: %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.InvoiceID != nil && *d.InvoiceID == inv.ID &&
			(d.Type == DiscrepancyMissingPaymentTx || d.Type == DiscrepancyPaymentAmountMismatch) {
			t.Fatalf("false payment discrepancy on healthy cycle books: %s %+v", d.Type, d)
		}
	}

	// --- Pre-fix corruption is now DETECTABLE: delete the final settle leg
	// (the shape the old dedup produced) and the reconciler must flag it. ---
	if _, err := conn.ExecContext(ctx,
		`DELETE FROM ledger_transactions WHERE reference_id=$1 AND code=3 AND occurrence=2`, inv.ID); err != nil {
		t.Fatalf("simulate corruption: %v", err)
	}
	report2, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("reconciliation Run (corrupted): %v", err)
	}
	flagged := false
	for _, d := range report2.Discrepancies {
		if d.InvoiceID != nil && *d.InvoiceID == inv.ID &&
			(d.Type == DiscrepancyMissingPaymentTx || d.Type == DiscrepancyPaymentAmountMismatch) {
			flagged = true
		}
	}
	if !flagged {
		t.Fatal("the swallowed-re-settle corruption shape was not flagged — the reconciler is still blind to it")
	}
}

// TestLedgerResettlement_WalletFunded_Postgres proves the wallet-funded
// settle→return→re-collect cycle keeps AR correct (audit finding A-ledger). A
// prepaid wallet drain settles part of the invoice at generation; the card
// settles the rest. When an ACH/card RETURN reverses only the cash, the reopened
// invoice must RETAIN the wallet portion as paid — reconstructed exactly as
// ReverseSettledPayment now does, via LatestSettledCashAmount — so re-collection
// posts the cash leg on (Total − wallet), not the full Total. Passing 0 (the
// pre-fix behaviour, from ReverseToUnpaid zeroing amount_paid) would post cash on
// the full Total and drive AR to −wallet.
func TestLedgerResettlement_WalletFunded_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed wallet resettlement test")
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

	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme', $4, NOW(), NOW())`,
		customerID, tenantID, "wal-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))

	const total = int64(100000)
	const wallet = int64(30000)
	const cash = total - wallet // 70000

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
		InvoiceNumber: "WAL-" + run, Total: total, Currency: "USD",
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',$4,$5,$5,'paid',$6,NOW(),NOW())`,
		inv.ID, tenantID, customerID, total, wallet, inv.InvoiceNumber); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	// Wallet drain at generation relieves AR by the wallet portion.
	if _, err := ledger.RecordWalletDrain(ctx, tenantID, nil, customerID, inv.ID, wallet, "wallet drain"); err != nil {
		t.Fatalf("RecordWalletDrain: %v", err)
	}

	// Settle #1: the card covers the remainder (cash = total − wallet).
	if err := ledger.RecordPaymentWithSettled(ctx, inv, wallet); err != nil {
		t.Fatalf("settle 1: %v", err)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeAR); b != 0 {
		t.Fatalf("after settle: AR = %d, want 0", b)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCash); b != cash {
		t.Fatalf("after settle: Cash = %d, want %d", b, cash)
	}

	// Bank returns the cash payment: reversal reinstates AR, removes Cash.
	if err := ledger.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse: %v", err)
	}

	// THE FIX: reconstruct the retained non-cash (wallet) portion from the actual
	// reversed cash leg — exactly what ReverseSettledPayment now does.
	cashAmt, err := ledger.LatestSettledCashAmount(ctx, inv.ID)
	if err != nil {
		t.Fatalf("LatestSettledCashAmount: %v", err)
	}
	retain := inv.Total - inv.CreditApplied - inv.TDSAmount - cashAmt
	if retain != wallet {
		t.Fatalf("reconstructed retained non-cash = %d, want %d (the wallet portion)", retain, wallet)
	}

	// Re-collect using the retained wallet portion: the cash leg posts on
	// (total − wallet), so AR nets back to 0. Passing 0 here — the pre-fix
	// behaviour — would post cash on the full total and leave AR at −wallet.
	if err := ledger.RecordPaymentWithSettled(ctx, inv, retain); err != nil {
		t.Fatalf("re-settle: %v", err)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeAR); b != 0 {
		t.Errorf("after re-collect: AR = %d, want 0 (the pre-fix bug drives it to -%d)", b, wallet)
	}
	if b := acctBalance(t, conn, tenantID, domain.AccountCodeCash); b != cash {
		t.Errorf("after re-collect: Cash = %d, want %d (one net settlement of the card portion)", b, cash)
	}
}
