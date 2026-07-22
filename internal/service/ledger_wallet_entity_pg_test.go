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

// TestLedger_WalletPerEntity_Postgres proves Multi-Entity Books Inc 2d: a
// wallet top-up and drain for a non-primary entity post to THAT entity's
// ledger (its own Cash/Customer-Credit/AR accounts), not the primary's.
// RecordWalletTopUp/RecordWalletDrain resolve the entity from the entityID the
// wallet service passes (the wallet's entity_id).
func TestLedger_WalletPerEntity_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed wallet-entity test")
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
		tenantID, "WE-"+run, "we-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := db.NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)

	customerID := uuid.New()
	walletTxID := uuid.New()
	invoiceID := uuid.New()

	// Top-up on the second entity: DR Cash / CR Customer Credit on ledger 2.
	if _, err := ledger.RecordWalletTopUp(ctx, tenantID, &second.ID, walletTxID, 80000, "top-up UK-"+run); err != nil {
		t.Fatalf("RecordWalletTopUp: %v", err)
	}
	// Drain against an invoice on the second entity: DR Customer Credit / CR AR.
	if _, err := ledger.RecordWalletDrain(ctx, tenantID, &second.ID, customerID, invoiceID, 30000, "drain UK-"+run); err != nil {
		t.Fatalf("RecordWalletDrain: %v", err)
	}

	// Both postings land on the second entity's ledger (id 2).
	for _, tc := range []struct {
		code uint16
		ref  uuid.UUID
		what string
	}{
		{domain.LedgerCodeWalletTopUp, walletTxID, "top-up"},
		{domain.LedgerCodeWalletDrain, invoiceID, "drain"},
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

	// The Customer Credit account it hit is tagged to the second entity.
	var creditEntity uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT entity_id FROM ledger_accounts WHERE tenant_id=$1 AND entity_id=$2 AND code=$3`,
		tenantID, second.ID, domain.AccountCodeCustomerCredit).Scan(&creditEntity); err != nil {
		t.Fatalf("second entity Customer Credit account missing: %v", err)
	}
	if creditEntity != second.ID {
		t.Errorf("Customer Credit account entity = %s, want %s", creditEntity, second.ID)
	}

	// The drain's AR leg is the entity-derived sub-ledger id, isolated from the
	// primary entity's per-customer AR (which uses the customer id directly).
	ent := ledgerEntity{ID: second.ID, LedgerID: 2}
	wantAR := ledger.arAccountID(ent, customerID)
	if wantAR == customerID {
		t.Fatalf("non-primary AR id must differ from the customer id")
	}
	var arCredited uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT credit_account_id FROM ledger_transactions WHERE reference_id=$1 AND code=$2`,
		invoiceID, domain.LedgerCodeWalletDrain).Scan(&arCredited); err != nil {
		t.Fatalf("drain AR leg not found: %v", err)
	}
	if arCredited != wantAR {
		t.Errorf("drain credited AR %s, want entity-scoped %s", arCredited, wantAR)
	}
}
