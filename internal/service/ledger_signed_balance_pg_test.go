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

// TestLedgerSignedBalance_Postgres proves the ENG-148 fix: ledger_accounts.balance
// is a signed double-entry balance (not accumulate-only). After a subscription
// invoice (DR AR / CR Deferred) and full recognition (DR Deferred / CR Recognized):
//   - AR (asset)        nets +amount
//   - Deferred (liab.)  nets to 0 (credited then debited the same amount)
//   - Recognized (rev.) nets +amount
//
// and the trial balance balances (sum of debits_posted == sum of credits_posted).
func TestLedgerSignedBalance_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed signed-balance test")
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
		tenantID, "Ledger-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := db.NewLedgerRepository(conn)
	svc := NewLedgerService(nil, repo)

	const amount int64 = 120000

	// Subscription invoice: DR AR (1100) / CR Deferred (2100).
	subID := uuid.New()
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		SubscriptionID: &subID, InvoiceNumber: "SUB-BAL-1", Total: amount, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}

	// Full recognition: DR Deferred (2100) / CR Recognized (4100).
	if _, err := svc.RecordRecognition(ctx, tenantID, amount, uuid.New()); err != nil {
		t.Fatalf("RecordRecognition: %v", err)
	}

	bal := func(code int) *domain.LedgerAccount {
		a, err := repo.GetAccountByTenantAndCode(ctx, tenantID, code)
		if err != nil || a == nil {
			t.Fatalf("GetAccountByTenantAndCode(%d): acct=%v err=%v", code, a, err)
		}
		return a
	}

	ar := bal(domain.AccountCodeAR)
	if ar.Balance != amount {
		t.Errorf("AR (asset) balance = %d, want %d (debit-normal, DR once)", ar.Balance, amount)
	}
	if ar.DebitsPosted != uint64(amount) || ar.CreditsPosted != 0 {
		t.Errorf("AR posted = D%d/C%d, want D%d/C0", ar.DebitsPosted, ar.CreditsPosted, amount)
	}

	deferred := bal(domain.AccountCodeDeferredRevenue)
	if deferred.Balance != 0 {
		t.Errorf("Deferred (liability) balance = %d, want 0 (credited then fully recognized)", deferred.Balance)
	}
	if deferred.CreditsPosted != uint64(amount) || deferred.DebitsPosted != uint64(amount) {
		t.Errorf("Deferred posted = D%d/C%d, want D%d/C%d", deferred.DebitsPosted, deferred.CreditsPosted, amount, amount)
	}

	recognized := bal(domain.AccountCodeRecognizedRevenue)
	if recognized.Balance != amount {
		t.Errorf("Recognized (revenue) balance = %d, want %d (credit-normal, CR once)", recognized.Balance, amount)
	}

	// Trial balance must balance: total debits == total credits across accounts.
	var totalD, totalC int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(debits_posted),0), COALESCE(SUM(credits_posted),0) FROM ledger_accounts WHERE tenant_id = $1`,
		tenantID).Scan(&totalD, &totalC); err != nil {
		t.Fatalf("sum posted: %v", err)
	}
	if totalD != totalC {
		t.Errorf("trial balance does not balance: total debits %d != total credits %d", totalD, totalC)
	}
	// Two postings of `amount` on each side (invoice + recognition).
	if totalD != 2*amount {
		t.Errorf("total debits = %d, want %d", totalD, 2*amount)
	}
}

// TestLedgerBackfill_Postgres proves the 000080 backfill SQL corrects a
// legacy accumulate-only row (balance = gross both-sides, posted columns 0) into
// signed balances rebuilt from the transaction log.
func TestLedgerBackfill_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed ledger backfill test")
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
		tenantID, "Backfill-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	// Three accounts seeded with the OLD accumulate-only balance (both sides of
	// every posting added to balance) and zeroed posted columns.
	arID, defID, recID := uuid.New(), uuid.New(), uuid.New()
	seedAcct := func(id uuid.UUID, name, typ string, code int, badBalance int64) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO ledger_accounts (id, tenant_id, name, type, code, ledger_id, currency, debits_posted, credits_posted, balance)
			 VALUES ($1, $2, $3, $4, $5, 1, 'USD', 0, 0, $6)`,
			id, tenantID, name, typ, code, badBalance); err != nil {
			t.Fatalf("seed account %s: %v", name, err)
		}
	}
	// AR (asset): DR 1000 once → legacy gross balance 1000.
	seedAcct(arID, "Accounts Receivable", "1", domain.AccountCodeAR, 1000)
	// Deferred (liability): CR 1000 then DR 400 → legacy accumulate-only 1400.
	seedAcct(defID, "Deferred Revenue", "2", domain.AccountCodeDeferredRevenue, 1400)
	// Recognized (revenue): CR 400 → legacy gross balance 400.
	seedAcct(recID, "Recognized Revenue", "4", domain.AccountCodeRecognizedRevenue, 400)

	// Real double-entry postings (both account ids are NOT NULL).
	seedTx := func(dr, cr uuid.UUID, amt int64, code int) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, created_at)
			 VALUES ($1, $2, $3, $4, 1, $5, NOW())`,
			uuid.New(), dr, cr, amt, code); err != nil {
			t.Fatalf("seed tx: %v", err)
		}
	}
	seedTx(arID, defID, 1000, 1) // invoice:     DR AR      / CR Deferred
	seedTx(defID, recID, 400, 2) // recognition: DR Deferred / CR Recognized

	// Run the 000080 backfill statements (mirrors the migration file).
	if _, err := conn.ExecContext(ctx, `UPDATE ledger_accounts la SET
		debits_posted  = COALESCE((SELECT SUM(amount) FROM ledger_transactions WHERE debit_account_id  = la.id), 0),
		credits_posted = COALESCE((SELECT SUM(amount) FROM ledger_transactions WHERE credit_account_id = la.id), 0)
		WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("backfill posted totals: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `UPDATE ledger_accounts SET
		balance = CASE WHEN lower(type) IN ('1','5','asset','expense')
		               THEN debits_posted - credits_posted
		               ELSE credits_posted - debits_posted END
		WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("backfill balance: %v", err)
	}

	repo := db.NewLedgerRepository(conn)
	ar, _ := repo.GetAccountByTenantAndCode(ctx, tenantID, domain.AccountCodeAR)
	if ar.Balance != 1000 { // asset: 1000 debits - 0 credits
		t.Errorf("AR balance after backfill = %d, want 1000", ar.Balance)
	}
	deferred, _ := repo.GetAccountByTenantAndCode(ctx, tenantID, domain.AccountCodeDeferredRevenue)
	if deferred.Balance != 600 { // liability: 1000 credits - 400 debits
		t.Errorf("Deferred balance after backfill = %d, want 600 (was accumulate-only 1400)", deferred.Balance)
	}
}
