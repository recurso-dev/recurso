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

// TestDunningHistory_TenantIsolation_Postgres verifies that GetHistoryStats and
// GetRecentHistory are scoped to a single tenant (ENG-157). Before the fix both
// aggregated dunning_history across ALL tenants, so one tenant's Smart Dunning
// dashboard reflected everyone's retries/successes. The test seeds two tenants
// with deliberately different counts and asserts neither leaks into the other.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database.
func TestDunningHistory_TenantIsolation_Postgres(t *testing.T) {
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

	repo := NewDunningRepository(conn)
	ctx := context.Background()
	tenantA, tenantB := uuid.New(), uuid.New()
	now := time.Now().UTC()

	// dunning_history.tenant_id is FK-enforced, so create the tenant rows.
	mkTenant := func(id uuid.UUID, name string) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, default_currency, email, created_at, updated_at)
			 VALUES ($1,$2,'USD',$3, now(), now()) ON CONFLICT (id) DO NOTHING`,
			id, name, name+"@example.com"); err != nil {
			t.Fatalf("create tenant: %v", err)
		}
	}
	mkTenant(tenantA, "TenantA")
	mkTenant(tenantB, "TenantB")

	// dunning_history.invoice_id is FK-enforced; a minimal invoice per tenant
	// suffices (customer_id is nullable, so no customer/ledger chain needed).
	mkInvoice := func(tenant uuid.UUID, num string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, status, created_at)
			 VALUES ($1,$2,'USD',1000,1000,$3,'open', now())`,
			id, tenant, num); err != nil {
			t.Fatalf("create invoice: %v", err)
		}
		return id
	}
	invA := mkInvoice(tenantA, "INV-A-1")
	invB := mkInvoice(tenantB, "INV-B-1")

	rec := func(tenant, invID uuid.UUID, outcome string, i int) {
		if err := repo.RecordHistory(ctx, domain.DunningHistory{
			ID: uuid.New(), TenantID: tenant, InvoiceID: invID,
			ContextKey: "ctx_test", ActionID: "retry_1d", RetryInterval: 1,
			Outcome: outcome, Reward: 0, CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("record history: %v", err)
		}
	}
	// Tenant A: 3 success + 2 failure = 5 retries.
	for i := 0; i < 3; i++ {
		rec(tenantA, invA, "success", i)
	}
	for i := 0; i < 2; i++ {
		rec(tenantA, invA, "failure", i+3)
	}
	// Tenant B: 4 success + 1 failure — must never bleed into A.
	for i := 0; i < 4; i++ {
		rec(tenantB, invB, "success", i)
	}
	rec(tenantB, invB, "failure", 4)

	// Stats are tenant-scoped.
	retries, successes, err := repo.GetHistoryStats(ctx, tenantA)
	if err != nil {
		t.Fatalf("GetHistoryStats: %v", err)
	}
	if retries != 5 || successes != 3 {
		t.Errorf("tenant A stats = (%d retries, %d successes), want (5, 3) — tenant B leaked in?", retries, successes)
	}

	// Recent history is tenant-scoped.
	hist, err := repo.GetRecentHistory(ctx, tenantA, 100)
	if err != nil {
		t.Fatalf("GetRecentHistory: %v", err)
	}
	if len(hist) != 5 {
		t.Errorf("tenant A history = %d rows, want 5", len(hist))
	}
	for _, h := range hist {
		if h.TenantID != tenantA {
			t.Errorf("history list leaked a row from tenant %s into tenant A", h.TenantID)
		}
	}
}
