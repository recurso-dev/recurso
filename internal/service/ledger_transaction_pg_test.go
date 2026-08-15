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

// TestGetTransactionByID_Postgres proves a single posted transaction (journal
// entry) is independently addressable by id, that each leg now carries its
// ledger-account id (the connectivity fix that lets the frontend deep-link a
// leg to its account page), and that the read is tenant-scoped: another
// tenant's id, and an unknown id, both return (nil, nil) → 404 at the handler.
func TestGetTransactionByID_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed ledger-transaction test")
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

	seedTenant := func(tag string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
			id, tag+"-"+id.String()[:8], id.String()[:8]+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
		return id
	}

	tenantID := seedTenant("TX")
	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		InvoiceNumber: "TX-1", Total: 12300, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}

	// Find the posting we just made (DR AR / CR Revenue, 12300).
	rows, err := svc.GetJournalEntriesByReference(ctx, tenantID, inv.ID)
	if err != nil {
		t.Fatalf("GetJournalEntriesByReference: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one journal entry for the invoice")
	}
	txID := rows[0].TransactionID

	// Owned read: returns the row, with BOTH account ids populated.
	got, err := svc.GetTransactionByID(ctx, tenantID, txID)
	if err != nil {
		t.Fatalf("GetTransactionByID(owned): %v", err)
	}
	if got == nil {
		t.Fatal("GetTransactionByID(owned) returned nil, want the transaction")
	}
	if got.TransactionID != txID {
		t.Errorf("transaction id = %s, want %s", got.TransactionID, txID)
	}
	if got.DebitAccountID == uuid.Nil || got.CreditAccountID == uuid.Nil {
		t.Errorf("account ids not populated: debit=%s credit=%s (leg deep-linking would be dead)",
			got.DebitAccountID, got.CreditAccountID)
	}
	// The projected ids must match the accounts named on the same row.
	if got.DebitAccountName == "" || got.CreditAccountName == "" {
		t.Errorf("account names missing: debit=%q credit=%q", got.DebitAccountName, got.CreditAccountName)
	}

	// Cross-tenant read: a different tenant asking for this id gets nothing.
	other := seedTenant("TXother")
	foreign, err := svc.GetTransactionByID(ctx, other, txID)
	if err != nil {
		t.Fatalf("GetTransactionByID(cross-tenant): %v", err)
	}
	if foreign != nil {
		t.Error("cross-tenant read returned a transaction; must be nil (404), never leak another tenant's ledger")
	}

	// Unknown id: nil, no error.
	missing, err := svc.GetTransactionByID(ctx, tenantID, uuid.New())
	if err != nil {
		t.Fatalf("GetTransactionByID(missing): %v", err)
	}
	if missing != nil {
		t.Error("unknown id returned a transaction, want nil")
	}
}
