package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestLedger_AccountCreditPerEntity_Postgres proves Multi-Entity Books Inc 2e:
// the account-credit issuance paths (adjustment credit, downgrade credit,
// downgrade tax reversal) post the Customer-Credit liability on the issuing
// entity's ledger, not the primary's. Each resolves the entity from the
// entityID the credit-note/subscription service passes.
func TestLedger_AccountCreditPerEntity_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed credit-entity test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "AC-"+run, "ac-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := db.NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)

	adjID, downID, taxID := uuid.New(), uuid.New(), uuid.New()
	if _, err := ledger.RecordAdjustmentCreditIssued(ctx, tenantID, &second.ID, adjID, 40000, "adj UK-"+run); err != nil {
		t.Fatalf("RecordAdjustmentCreditIssued: %v", err)
	}
	if _, err := ledger.RecordDowngradeCredit(ctx, tenantID, &second.ID, downID, 25000, "downgrade UK-"+run); err != nil {
		t.Fatalf("RecordDowngradeCredit: %v", err)
	}
	if _, err := ledger.RecordDowngradeTaxReversal(ctx, tenantID, &second.ID, taxID, 4500, "downgrade GST UK-"+run); err != nil {
		t.Fatalf("RecordDowngradeTaxReversal: %v", err)
	}

	// Each posting lands on the second entity's ledger (id 2), crediting its
	// own Customer-Credit account.
	for _, tc := range []struct {
		code uint16
		ref  uuid.UUID
		what string
	}{
		{8, adjID, "adjustment credit"},
		{domain.LedgerCodeDowngradeCredit, downID, "downgrade credit"},
		{domain.LedgerCodeDowngradeTaxReversal, taxID, "downgrade tax reversal"},
	} {
		var ledgerID int
		if err := conn.QueryRowContext(ctx,
			`SELECT ledger_id FROM ledger_transactions WHERE reference_id=$1 AND code=$2`, tc.ref, tc.code).Scan(&ledgerID); err != nil {
			t.Fatalf("%s posting not found: %v", tc.what, err)
		}
		if ledgerID != 2 {
			t.Errorf("%s posted to ledger %d, want the entity's ledger 2", tc.what, ledgerID)
		}
	}

	// The Customer Credit account they credit is tagged to the second entity,
	// isolated from the primary's Customer-Credit GL.
	var creditEntity uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT entity_id FROM ledger_accounts WHERE tenant_id=$1 AND entity_id=$2 AND code=$3`,
		tenantID, second.ID, domain.AccountCodeCustomerCredit).Scan(&creditEntity); err != nil {
		t.Fatalf("second entity Customer Credit account missing: %v", err)
	}
	if creditEntity != second.ID {
		t.Errorf("Customer Credit account entity = %s, want %s", creditEntity, second.ID)
	}
}
