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

// TestInvoiceRepository_ListPaginated proves the API list path is bounded and
// pages correctly (newest first, no overlap, no gaps), and that CountByTenant
// reports the true total. This is the fix for the previously-unbounded
// GET /v1/invoices.
func TestInvoiceRepository_ListPaginated(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed pagination test")
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

	// 5 invoices, each newer than the last, so ordering is deterministic.
	base := time.Now().Add(-time.Hour)
	ids := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		inv := &domain.Invoice{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CustomerID:    customerID,
			InvoiceNumber: "PG-" + uuid.New().String()[:8],
			Currency:      "USD",
			Subtotal:      1000,
			Total:         1000,
			Status:        domain.InvoiceStatusOpen,
			CreatedAt:     base.Add(time.Duration(i) * time.Minute),
			DueDate:       base.Add(time.Duration(i) * time.Minute),
		}
		if err := repo.Create(ctx, inv); err != nil {
			t.Fatalf("create invoice %d: %v", i, err)
		}
		ids[i] = inv.ID
	}

	total, err := repo.CountByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("CountByTenant: %v", err)
	}
	if total != 5 {
		t.Fatalf("CountByTenant = %d, want 5", total)
	}

	// Page through with limit 2: pages of 2, 2, 1.
	seen := map[uuid.UUID]bool{}
	var pageSizes []int
	var prevOldest time.Time
	for offset := 0; offset < 6; offset += 2 {
		page, err := repo.ListPaginated(ctx, tenantID, 2, offset)
		if err != nil {
			t.Fatalf("ListPaginated(offset=%d): %v", offset, err)
		}
		if len(page) == 0 {
			break
		}
		pageSizes = append(pageSizes, len(page))
		for i, inv := range page {
			if seen[inv.ID] {
				t.Errorf("invoice %s appeared on two pages (overlap)", inv.ID)
			}
			seen[inv.ID] = true
			// Newest-first within and across pages.
			if i > 0 && inv.CreatedAt.After(page[i-1].CreatedAt) {
				t.Errorf("page not sorted newest-first at index %d", i)
			}
			if !prevOldest.IsZero() && inv.CreatedAt.After(prevOldest) {
				t.Errorf("page ordering not monotonic across pages")
			}
		}
		prevOldest = page[len(page)-1].CreatedAt
	}

	if len(seen) != 5 {
		t.Errorf("paging covered %d distinct invoices, want 5", len(seen))
	}
	if len(pageSizes) != 3 || pageSizes[0] != 2 || pageSizes[1] != 2 || pageSizes[2] != 1 {
		t.Errorf("page sizes = %v, want [2 2 1]", pageSizes)
	}
}
