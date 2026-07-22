package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// newSeriesInvoice builds a minimal open invoice with one line item (so it goes
// through the atomic tx-insert path) and NO invoice number, forcing the repo to
// allocate one from the issuing entity's gapless series.
func newSeriesInvoice(tenantID, customerID uuid.UUID, entityID *uuid.UUID) *domain.Invoice {
	id := uuid.New()
	return &domain.Invoice{
		ID:         id,
		TenantID:   tenantID,
		EntityID:   entityID,
		CustomerID: customerID,
		Status:     domain.InvoiceStatusOpen,
		Currency:   "USD",
		Subtotal:   1000,
		Total:      1000,
		CreatedAt:  time.Now(),
		DueDate:    time.Now(),
		LineItems: []domain.InvoiceItem{
			{ID: uuid.New(), Description: "line", Quantity: 1, UnitAmount: 1000, Amount: 1000},
		},
	}
}

// TestInvoiceSeries_PerEntityGapless_Postgres proves Multi-Entity Books Inc 3a:
// each entity issues its own gapless {prefix}-{seq:06d} series, a nil entity
// draws the tenant's primary series, and the two series are independent.
func TestInvoiceSeries_PerEntityGapless_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invoice-series test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	// A second entity with its own prefix and series.
	entityRepo := NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	repo := NewInvoiceRepository(conn).(*InvoiceRepository)

	// The primary series (nil entity) numbers INV-000001, INV-000002, …
	for i, want := range []string{"INV-000001", "INV-000002"} {
		inv := newSeriesInvoice(tenantID, customerID, nil)
		if err := repo.Create(ctx, inv); err != nil {
			t.Fatalf("create primary invoice %d: %v", i, err)
		}
		if inv.InvoiceNumber != want {
			t.Errorf("primary invoice %d number = %q, want %q", i, inv.InvoiceNumber, want)
		}
	}

	// The second entity has an independent series starting at 1.
	inv := newSeriesInvoice(tenantID, customerID, &second.ID)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("create second-entity invoice: %v", err)
	}
	if inv.InvoiceNumber != "UK-000001" {
		t.Errorf("second-entity invoice number = %q, want UK-000001", inv.InvoiceNumber)
	}

	// A caller-supplied number is never overwritten (imports, backfills).
	explicit := newSeriesInvoice(tenantID, customerID, nil)
	explicit.InvoiceNumber = "LEGACY-42"
	if err := repo.Create(ctx, explicit); err != nil {
		t.Fatalf("create explicit-number invoice: %v", err)
	}
	if explicit.InvoiceNumber != "LEGACY-42" {
		t.Errorf("explicit number was overwritten to %q", explicit.InvoiceNumber)
	}
}

// TestInvoiceSeries_ConcurrentGapless_Postgres proves the series is gapless and
// unique under concurrency: N invoices created in parallel for one entity draw
// exactly seq 1..N with no gap or duplicate (the UPDATE…RETURNING serializes on
// the entity's sequence row).
func TestInvoiceSeries_ConcurrentGapless_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invoice-series test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	entityRepo := NewEntityRepository(conn)
	ent := &domain.Entity{TenantID: tenantID, Name: "Concurrent Co", InvoicePrefix: "CC"}
	if err := entityRepo.Create(ctx, ent); err != nil {
		t.Fatalf("create entity: %v", err)
	}
	repo := NewInvoiceRepository(conn).(*InvoiceRepository)

	const n = 25
	numbers := make([]string, n)
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			inv := newSeriesInvoice(tenantID, customerID, &ent.ID)
			if err := repo.Create(ctx, inv); err != nil {
				errs <- err
				return
			}
			numbers[i] = inv.InvoiceNumber
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent create: %v", err)
	}

	// Every seq 1..N appears exactly once — no gap, no duplicate.
	seen := make(map[string]bool, n)
	for _, num := range numbers {
		if seen[num] {
			t.Errorf("duplicate invoice number %q", num)
		}
		seen[num] = true
	}
	for i := 1; i <= n; i++ {
		want := fmt.Sprintf("CC-%06d", i)
		if !seen[want] {
			t.Errorf("missing invoice number %q (series has a gap)", want)
		}
	}
}
