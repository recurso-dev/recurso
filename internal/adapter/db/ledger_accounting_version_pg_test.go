package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestLedgerAccountingVersion_Postgres proves journal-level accounting-model
// provenance (ADR-008 increment 2): a journal is stamped with the deployment's
// active model (V1 cash by default, V2 when accrual is wired), and a transaction
// carrying its own version overrides that. So an auditor can always ask "which
// accounting rules produced THIS journal entry?".
func TestLedgerAccountingVersion_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed ledger-version test")
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

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "LV-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewLedgerRepository(conn)
	seedAccount := func(code int) uuid.UUID {
		id := uuid.New()
		if err := repo.CreateAccount(ctx, &domain.LedgerAccount{
			ID: id, TenantID: tenantID, Name: "acct", Type: domain.AccountTypeAsset,
			Code: code, LedgerID: 1, Currency: "USD",
		}); err != nil {
			t.Fatalf("seed account %d: %v", code, err)
		}
		return id
	}
	dr, cr := seedAccount(1000), seedAccount(4000)

	versionOf := func(txID uuid.UUID) int {
		var v int
		if err := conn.QueryRowContext(ctx,
			`SELECT accounting_version FROM ledger_transactions WHERE id = $1`, txID).Scan(&v); err != nil {
			t.Fatalf("read accounting_version: %v", err)
		}
		return v
	}

	post := func(tx *domain.LedgerTransaction) uuid.UUID {
		tx.ID = uuid.New()
		tx.DebitAccountID, tx.CreditAccountID = dr, cr
		tx.Amount = 1000
		tx.LedgerID = 1
		tx.Code = 1
		tx.ReferenceID = uuid.New()
		if err := repo.CreateTransaction(ctx, tx); err != nil {
			t.Fatalf("post tx: %v", err)
		}
		return tx.ID
	}

	// Default repository → cash model (V1).
	if got := versionOf(post(&domain.LedgerTransaction{})); got != domain.AccountingModelV1 {
		t.Errorf("default journal version = %d, want %d (V1 cash)", got, domain.AccountingModelV1)
	}

	// Deployment configured for accrual → every journal stamped V2.
	repo.SetAccountingVersion(domain.AccountingModelV2)
	if got := versionOf(post(&domain.LedgerTransaction{})); got != domain.AccountingModelV2 {
		t.Errorf("accrual-deployment journal version = %d, want %d (V2)", got, domain.AccountingModelV2)
	}

	// A per-posting override on the transaction wins over the deployment default.
	repo.SetAccountingVersion(domain.AccountingModelV1)
	if got := versionOf(post(&domain.LedgerTransaction{AccountingVersion: domain.AccountingModelV2})); got != domain.AccountingModelV2 {
		t.Errorf("per-tx override version = %d, want %d (V2)", got, domain.AccountingModelV2)
	}
}
