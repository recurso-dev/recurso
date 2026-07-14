package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestInvoiceRepository_BillingReasonRoundTrips proves the billing_reason the
// API/SDK document is now actually persisted and read back through both the
// single-invoice and list read paths (it was previously a dead field — the
// column didn't exist and the insert never wrote it).
func TestInvoiceRepository_BillingReasonRoundTrips(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed billing_reason test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo := NewInvoiceRepository(conn)
	ctx := context.Background()

	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	inv := &domain.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		CustomerID:    customerID,
		InvoiceNumber: "BR-" + uuid.New().String()[:8],
		BillingReason: domain.BillingReasonMandateDebit,
		Currency:      "INR",
		Subtotal:      10000,
		Total:         10000,
		Status:        domain.InvoiceStatusOpen,
		CreatedAt:     time.Now(),
		DueDate:       time.Now(),
	}
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	// Single-invoice read path (GET /invoices/:id).
	got, err := repo.GetByIDPublic(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByIDPublic: %v", err)
	}
	if got.BillingReason != domain.BillingReasonMandateDebit {
		t.Errorf("GetByID billing_reason = %q, want %q", got.BillingReason, domain.BillingReasonMandateDebit)
	}

	// List read path — the invoice must carry its reason there too.
	list, err := repo.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, li := range list {
		if li.ID == inv.ID {
			found = true
			if li.BillingReason != domain.BillingReasonMandateDebit {
				t.Errorf("List billing_reason = %q, want %q", li.BillingReason, domain.BillingReasonMandateDebit)
			}
		}
	}
	if !found {
		t.Fatal("created invoice not returned by List")
	}
}
