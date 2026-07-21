package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// The resolved tax type is persisted at invoice creation and readable via a
// direct column query (Track D · D3c) — the audit foundation for the exempt
// breakout. Exercises the real INSERT (invoiceInsertQuery + insertInvoiceRow).
func TestInvoiceTaxType_PersistedOnCreate_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invoice tax-type test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "TT-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, "c@x.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	repo := NewInvoiceRepository(conn)
	inv := &domain.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		CustomerID:    custID,
		InvoiceNumber: "INV-TT-1",
		Status:        domain.InvoiceStatusOpen,
		Currency:      "USD",
		Subtotal:      50000,
		TaxAmount:     0,
		TaxType:       "sales_tax_exempt",
		Total:         50000,
		CreatedAt:     time.Now().UTC(),
		DueDate:       time.Now().UTC(),
	}
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	var got string
	if err := conn.QueryRowContext(ctx,
		`SELECT tax_type FROM invoices WHERE id = $1`, inv.ID).Scan(&got); err != nil {
		t.Fatalf("read tax_type: %v", err)
	}
	if got != "sales_tax_exempt" {
		t.Errorf("persisted tax_type = %q, want sales_tax_exempt", got)
	}

	// An invoice created without a tax type defaults to '' (the column default).
	plain := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: custID, InvoiceNumber: "INV-TT-2",
		Status: domain.InvoiceStatusOpen, Currency: "USD", Subtotal: 100, Total: 100,
		CreatedAt: time.Now().UTC(), DueDate: time.Now().UTC(),
	}
	if err := repo.Create(ctx, plain); err != nil {
		t.Fatalf("create plain invoice: %v", err)
	}
	if err := conn.QueryRowContext(ctx,
		`SELECT tax_type FROM invoices WHERE id = $1`, plain.ID).Scan(&got); err != nil {
		t.Fatalf("read plain tax_type: %v", err)
	}
	if got != "" {
		t.Errorf("default tax_type = %q, want empty", got)
	}
}
