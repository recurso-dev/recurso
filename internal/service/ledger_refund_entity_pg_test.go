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

// TestLedger_RefundPerEntity_Postgres proves Multi-Entity Books Inc 2c: a refund
// for a non-primary entity's credit note posts to THAT entity's ledger (its own
// Refunds/Cash accounts), not the primary's. RecordRefund resolves the entity
// from the entityID the credit-note service passes (inv.EntityID).
func TestLedger_RefundPerEntity_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed refund-entity test")
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
		tenantID, "RF-"+run, "rf-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := db.NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)

	cnID := uuid.New()
	// Refund for the second entity (entityID = second.ID).
	if err := ledger.RecordRefund(ctx, tenantID, &second.ID, cnID, 50000, "refund UK-"+run); err != nil {
		t.Fatalf("RecordRefund: %v", err)
	}

	// The refund posting is on the second entity's ledger (id 2).
	var ledgerID int
	if err := conn.QueryRowContext(ctx,
		`SELECT ledger_id FROM ledger_transactions WHERE reference_id = $1 AND code = 4`, cnID).Scan(&ledgerID); err != nil {
		t.Fatalf("refund posting not found: %v", err)
	}
	if ledgerID != 2 {
		t.Errorf("refund posted to ledger %d, want the entity's ledger 2", ledgerID)
	}
	// The Refunds account it hit is tagged to the second entity.
	var refundsEntity uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT entity_id FROM ledger_accounts WHERE tenant_id=$1 AND entity_id=$2 AND code=$3`,
		tenantID, second.ID, domain.AccountCodeRefunds).Scan(&refundsEntity); err != nil {
		t.Fatalf("second entity Refunds account missing: %v", err)
	}
	if refundsEntity != second.ID {
		t.Errorf("Refunds account entity = %s, want %s", refundsEntity, second.ID)
	}
}
