package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestCountActiveByCustomer_Postgres proves the ENG-152 backend count: only
// active subscriptions are counted, per customer, and customers with none are
// absent from the map.
func TestCountActiveByCustomer_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed active-subs count test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Subs-"+run, "subs-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	custA, custB := uuid.New(), uuid.New()
	for _, cid := range []uuid.UUID{custA, custB} {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
			cid, tenantID, cid.String()[:8]+"@t.com", uuid.New()); err != nil {
			t.Fatalf("seed customer: %v", err)
		}
	}

	// Customer A: 2 active + 1 canceled. Customer B: 1 canceled (0 active).
	seedSub := func(cid uuid.UUID, status string) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NOW(), NOW() + INTERVAL '1 month', NOW(), NOW(), NOW())`,
			uuid.New(), tenantID, cid, planID, status); err != nil {
			t.Fatalf("seed subscription: %v", err)
		}
	}
	seedSub(custA, "active")
	seedSub(custA, "active")
	seedSub(custA, "canceled")
	seedSub(custB, "canceled")

	repo := &SubscriptionRepository{db: conn}
	counts, err := repo.CountActiveByCustomer(ctx, tenantID)
	if err != nil {
		t.Fatalf("CountActiveByCustomer: %v", err)
	}
	if counts[custA] != 2 {
		t.Errorf("customer A active count = %d, want 2 (canceled excluded)", counts[custA])
	}
	if _, ok := counts[custB]; ok {
		t.Errorf("customer B present with count %d, want absent (no active subs)", counts[custB])
	}
}
