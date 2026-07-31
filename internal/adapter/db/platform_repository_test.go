package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestPlatformMetrics_Postgres seeds a couple of tenants (one activated with a
// customer and a trial expiring soon, one dormant) and checks the cross-tenant
// aggregation. Uses >= assertions on the shared DB counts, and exact checks on
// the specific seeded rows surfaced in RecentSignups.
func TestPlatformMetrics_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping platform metrics PG test")
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
	activatedID := uuid.New()
	dormantID := uuid.New()

	// Activated tenant: trialing, trial expiring in 3 days, and it created a customer.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, billing_status, plan_tier, trial_ends_at)
		 VALUES ($1, $2, $3, NOW(), 'trialing', 'trial', NOW() + INTERVAL '3 days')`,
		activatedID, "Acme-"+run, "acme-"+run+"@t.com"); err != nil {
		t.Fatalf("seed activated tenant: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		uuid.New(), activatedID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	// Dormant tenant: active plan, no customer.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, billing_status, plan_tier)
		 VALUES ($1, $2, $3, NOW(), 'active', 'free')`,
		dormantID, "Dormant-"+run, "dorm-"+run+"@t.com"); err != nil {
		t.Fatalf("seed dormant tenant: %v", err)
	}

	m, err := NewPlatformRepository(conn).PlatformMetrics(ctx)
	if err != nil {
		t.Fatalf("PlatformMetrics: %v", err)
	}

	if m.TotalTenants < 2 {
		t.Errorf("total tenants = %d, want >= 2", m.TotalTenants)
	}
	if m.SignupsLast7d < 2 || m.SignupsLast30d < 2 {
		t.Errorf("signups 7d/30d = %d/%d, want >= 2 each", m.SignupsLast7d, m.SignupsLast30d)
	}
	if m.ActivatedTenants < 1 {
		t.Errorf("activated = %d, want >= 1", m.ActivatedTenants)
	}
	if m.TrialsExpiring7d < 1 {
		t.Errorf("trials expiring 7d = %d, want >= 1", m.TrialsExpiring7d)
	}
	if m.ByBillingStatus["trialing"] < 1 || m.ByPlanTier["trial"] < 1 {
		t.Errorf("breakdowns missing seeded rows: status=%+v tier=%+v", m.ByBillingStatus, m.ByPlanTier)
	}

	// The seeded rows must surface in RecentSignups with correct activation flags.
	var gotAct, gotDorm bool
	for i := range m.RecentSignups {
		s := m.RecentSignups[i]
		if s.Email == "acme-"+run+"@t.com" {
			gotAct = true
			if !s.Activated {
				t.Errorf("activated tenant should be flagged activated: %+v", s)
			}
			if s.TrialEndsAt == nil {
				t.Errorf("activated tenant should carry trial_ends_at: %+v", s)
			}
		}
		if s.Email == "dorm-"+run+"@t.com" {
			gotDorm = true
			if s.Activated {
				t.Errorf("dormant tenant must not be flagged activated: %+v", s)
			}
		}
	}
	if !gotAct || !gotDorm {
		t.Errorf("seeded tenants not both in recent signups (act=%v dorm=%v)", gotAct, gotDorm)
	}
}
