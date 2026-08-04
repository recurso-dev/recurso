package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestGeneralLedgerExport_CarriesAccountingVersion proves the accounting-model
// provenance stamped on each journal (ADR-008 increment 2) is surfaced in the
// general-ledger export an auditor pulls — GetGeneralLedgerRows returns the
// version, so it flows into the GL CSV/JSON.
func TestGeneralLedgerExport_CarriesAccountingVersion(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed GL-export version test")
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
		tenantID, "GLV-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
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

	// Post a journal stamped V2 (accrual).
	tx := &domain.LedgerTransaction{
		ID: uuid.New(), DebitAccountID: dr, CreditAccountID: cr, Amount: 1000,
		LedgerID: 1, Code: 1, ReferenceID: uuid.New(), AccountingVersion: domain.AccountingModelV2,
	}
	if err := repo.CreateTransaction(ctx, tx); err != nil {
		t.Fatalf("post tx: %v", err)
	}

	rows, err := repo.GetGeneralLedgerRows(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetGeneralLedgerRows: %v", err)
	}
	var found bool
	for _, g := range rows {
		if g.TransactionID == tx.ID {
			found = true
			if g.AccountingVersion != domain.AccountingModelV2 {
				t.Errorf("GL row accounting_version = %d, want %d (V2)", g.AccountingVersion, domain.AccountingModelV2)
			}
		}
	}
	if !found {
		t.Fatalf("posted journal %s not present in GL export", tx.ID)
	}
}
