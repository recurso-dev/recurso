package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestDunningTimingRates_Postgres proves GetTimingRates tallies success/total by
// hour-of-day and day-of-week from dunning_history (Collections Intelligence
// Inc 4).
func TestDunningTimingRates_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed dunning-timing test")
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
	// Pin UTC so EXTRACT(HOUR ...) is deterministic regardless of the local
	// session timezone (CI is UTC; a dev box may not be).
	if _, err := conn.ExecContext(ctx, `SET TIME ZONE 'UTC'`); err != nil {
		t.Fatalf("set utc: %v", err)
	}
	run := uuid.New().String()[:8]
	repo := NewDunningRepository(conn)

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Tim-"+run, "tim-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		custID, tenantID, "timc-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1, $2, $3, 'USD', 1000, 1000, 0, 0, 'past_due', $4, NOW(), NOW())`,
		invID, tenantID, custID, "INV-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	// A fixed UTC instant: 2026-03-04 09:00 is a Wednesday (DOW=3), hour=9.
	seed := func(outcome string, ts string) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO dunning_history (id, tenant_id, invoice_id, context_key, action_id, retry_interval, outcome, reward, created_at)
			 VALUES ($1, $2, $3, 'ctx', 'a1', 3600, $4, $5, $6::timestamptz)`,
			uuid.New(), tenantID, invID, outcome, map[string]float64{"success": 1, "failure": 0}[outcome], ts); err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}
	// Hour 9 (Wed): 2 success + 1 failure → 2/3.
	seed("success", "2026-03-04 09:15:00+00")
	seed("success", "2026-03-04 09:45:00+00")
	seed("failure", "2026-03-04 09:05:00+00")
	// Hour 14 (Wed): 1 failure.
	seed("failure", "2026-03-04 14:30:00+00")

	buckets, err := repo.GetTimingRates(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTimingRates: %v", err)
	}
	// Force UTC interpretation to match EXTRACT on the session (CI runs UTC).
	var hour9, hour14, dow3 *struct{ total, succ int }
	for _, b := range buckets {
		switch {
		case b.Unit == "hour" && b.Bucket == 9:
			hour9 = &struct{ total, succ int }{b.Total, b.Successes}
		case b.Unit == "hour" && b.Bucket == 14:
			hour14 = &struct{ total, succ int }{b.Total, b.Successes}
		case b.Unit == "dow" && b.Bucket == 3:
			dow3 = &struct{ total, succ int }{b.Total, b.Successes}
		}
	}
	if hour9 == nil || hour9.total != 3 || hour9.succ != 2 {
		t.Errorf("hour 9 = %+v, want total 3 succ 2", hour9)
	}
	if hour14 == nil || hour14.total != 1 || hour14.succ != 0 {
		t.Errorf("hour 14 = %+v, want total 1 succ 0", hour14)
	}
	if dow3 == nil || dow3.total != 4 || dow3.succ != 2 {
		t.Errorf("dow 3 (Wed) = %+v, want total 4 succ 2", dow3)
	}
}
