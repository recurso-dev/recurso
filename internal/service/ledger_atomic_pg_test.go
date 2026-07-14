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

// TestCreateTransactions_AtomicRollback_Postgres proves the multi-leg posting is
// all-or-nothing: a batch whose second leg violates the account FK leaves the
// first (valid) leg uncommitted too, so an invoice can never post its AR leg
// without its GST reclassification (or vice-versa).
func TestCreateTransactions_AtomicRollback_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed atomic-ledger test")
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
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Atomic-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := db.NewLedgerRepository(conn)
	cash := &domain.LedgerAccount{ID: uuid.New(), TenantID: tenantID, Name: "Cash", Type: domain.AccountTypeAsset, Code: domain.AccountCodeCash, LedgerID: 1, Currency: "USD"}
	rev := &domain.LedgerAccount{ID: uuid.New(), TenantID: tenantID, Name: "Revenue", Type: domain.AccountTypeRevenue, Code: domain.AccountCodeRevenue, LedgerID: 1, Currency: "USD"}
	if err := repo.CreateAccount(ctx, cash); err != nil {
		t.Fatalf("create cash: %v", err)
	}
	if err := repo.CreateAccount(ctx, rev); err != nil {
		t.Fatalf("create revenue: %v", err)
	}

	countByRef := func(ref uuid.UUID) int {
		var n int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1`, ref).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	// Happy path: two valid legs post together.
	okRef := uuid.New()
	if err := repo.CreateTransactions(ctx, []*domain.LedgerTransaction{
		{ID: uuid.New(), DebitAccountID: cash.ID, CreditAccountID: rev.ID, Amount: 1000, LedgerID: 1, Code: 1, ReferenceID: okRef, Description: "leg1"},
		{ID: uuid.New(), DebitAccountID: cash.ID, CreditAccountID: rev.ID, Amount: 500, LedgerID: 1, Code: 2, ReferenceID: okRef, Description: "leg2"},
	}); err != nil {
		t.Fatalf("CreateTransactions (valid): %v", err)
	}
	if got := countByRef(okRef); got != 2 {
		t.Fatalf("valid batch posted %d rows, want 2", got)
	}

	// Rollback path: first leg valid, second references a non-existent account
	// (FK violation). Neither should persist.
	badRef := uuid.New()
	err = repo.CreateTransactions(ctx, []*domain.LedgerTransaction{
		{ID: uuid.New(), DebitAccountID: cash.ID, CreditAccountID: rev.ID, Amount: 700, LedgerID: 1, Code: 3, ReferenceID: badRef, Description: "valid-first"},
		{ID: uuid.New(), DebitAccountID: uuid.New(), CreditAccountID: rev.ID, Amount: 300, LedgerID: 1, Code: 4, ReferenceID: badRef, Description: "fk-violation"},
	})
	if err == nil {
		t.Fatal("expected an FK error on the second leg, got nil")
	}
	if got := countByRef(badRef); got != 0 {
		t.Errorf("after a failed batch, %d rows persisted for the ref, want 0 (the valid first leg must roll back)", got)
	}
}
