package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestInvoiceItemRepository_Postgres exercises the real SQL for itemized invoice
// lines: bulk create, read-back, batch read, and cascade-delete when the parent
// invoice is removed. It applies the embedded migrations first, which validates
// migration 000070.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database, e.g.:
//
//	createdb recurso_repo_test
//	TEST_DATABASE_URL='postgres://localhost:5432/recurso_repo_test?sslmode=disable' go test ./internal/adapter/db/
func TestInvoiceItemRepository_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed repository test")
	}

	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx := context.Background()
	invoiceRepo := NewInvoiceRepository(conn)
	itemRepo := NewInvoiceItemRepository(conn)

	// A tenant + customer are needed to satisfy the invoice FKs.
	tenantID := seedTenantAndCustomer(t, conn)
	customerID := tenantID.customerID

	// Create an invoice carrying two line items — they must land atomically.
	invID := uuid.New()
	now := time.Now().UTC()
	inv := &domain.Invoice{
		ID:            invID,
		TenantID:      tenantID.tenantID,
		CustomerID:    customerID,
		InvoiceNumber: "INV-ITEMS-" + invID.String()[:8],
		Status:        domain.InvoiceStatusOpen,
		Currency:      "INR",
		Subtotal:      150000,
		TaxAmount:     27000,
		Total:         177000,
		IGSTAmount:    27000,
		CreatedAt:     now,
		DueDate:       now,
		LineItems: []domain.InvoiceItem{
			{Description: "Base", HSNCode: "998314", Quantity: 1, UnitAmount: 100000, Amount: 100000, TaxRate: 18, IGSTAmount: 18000, TaxableAmount: 100000},
			{Description: "Add-on", HSNCode: "998314", Quantity: 1, UnitAmount: 50000, Amount: 50000, TaxRate: 18, IGSTAmount: 9000, TaxableAmount: 50000},
		},
	}
	if err := invoiceRepo.Create(ctx, inv); err != nil {
		t.Fatalf("invoice create (with items) failed: %v", err)
	}

	// Read back the items directly.
	items, err := itemRepo.ListByInvoiceID(ctx, invID)
	if err != nil {
		t.Fatalf("ListByInvoiceID failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("read back %d items, want 2", len(items))
	}
	var sumAmount, sumTax int64
	for _, it := range items {
		if it.InvoiceID != invID {
			t.Errorf("item invoice_id = %s, want %s", it.InvoiceID, invID)
		}
		sumAmount += it.Amount
		sumTax += it.CGSTAmount + it.SGSTAmount + it.IGSTAmount
	}
	if sumAmount != inv.Subtotal {
		t.Errorf("Σ item amount = %d, want %d", sumAmount, inv.Subtotal)
	}
	if sumTax != inv.TaxAmount {
		t.Errorf("Σ item tax = %d, want %d", sumTax, inv.TaxAmount)
	}

	// GetByID hydrates line items onto the invoice.
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID.tenantID)
	got, err := invoiceRepo.GetByID(tctx, invID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil || len(got.LineItems) != 2 {
		t.Fatalf("GetByID hydrated %d line items, want 2", len(got.LineItems))
	}

	// Bulk Create (standalone) on a second invoice.
	inv2ID := uuid.New()
	inv2 := &domain.Invoice{
		ID: inv2ID, TenantID: tenantID.tenantID, CustomerID: customerID,
		InvoiceNumber: "INV-ITEMS2-" + inv2ID.String()[:8], Status: domain.InvoiceStatusOpen,
		Currency: "INR", Subtotal: 5000, Total: 5000, CreatedAt: now, DueDate: now,
	}
	if err := invoiceRepo.Create(ctx, inv2); err != nil {
		t.Fatalf("second invoice create failed: %v", err)
	}
	standalone := []*domain.InvoiceItem{
		{ID: uuid.New(), InvoiceID: inv2ID, Description: "One-off", HSNCode: "998314", Quantity: 1, UnitAmount: 5000, Amount: 5000, TaxableAmount: 5000, CreatedAt: now},
	}
	if err := itemRepo.Create(ctx, standalone); err != nil {
		t.Fatalf("standalone item Create failed: %v", err)
	}

	// Batch read across both invoices.
	batch, err := itemRepo.ListByInvoiceIDs(ctx, []uuid.UUID{invID, inv2ID})
	if err != nil {
		t.Fatalf("ListByInvoiceIDs failed: %v", err)
	}
	if len(batch[invID]) != 2 || len(batch[inv2ID]) != 1 {
		t.Fatalf("batch counts = %d/%d, want 2/1", len(batch[invID]), len(batch[inv2ID]))
	}

	// Cascade delete: removing the invoice removes its items.
	if _, err := conn.ExecContext(ctx, "DELETE FROM invoices WHERE id = $1", invID); err != nil {
		t.Fatalf("delete invoice failed: %v", err)
	}
	after, err := itemRepo.ListByInvoiceID(ctx, invID)
	if err != nil {
		t.Fatalf("ListByInvoiceID after delete failed: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("expected 0 items after cascade delete, got %d", len(after))
	}
}

type seededTenant struct {
	tenantID   uuid.UUID
	customerID uuid.UUID
}

// seedTenantAndCustomer inserts a minimal tenant + customer so invoice FKs hold.
func seedTenantAndCustomer(t *testing.T, conn *sql.DB) seededTenant {
	t.Helper()
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, created_at) VALUES ($1, $2, NOW())`,
		tenantID, "recon-test-tenant"); err != nil {
		t.Fatalf("seed tenant failed: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "recon-"+customerID.String()[:8]+"@example.com", uuid.New()); err != nil {
		t.Fatalf("seed customer failed: %v", err)
	}
	return seededTenant{tenantID: tenantID, customerID: customerID}
}
