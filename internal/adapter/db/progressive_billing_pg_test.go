package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openProgressiveTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed progressive-billing test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// seedProgressiveScope creates tenant/plan/metric/customer/subscription/charge
// so a watermark row can satisfy its foreign keys. Returns (tenantID, subID,
// chargeID).
func seedProgressiveScope(t *testing.T, conn *sql.DB) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Prog-"+run, "prog-"+run+"@t.com")
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "pro-"+run)
	metricID := uuid.New()
	must(t, conn, `INSERT INTO billable_metrics (id, tenant_id, name, code, aggregation_type, field_name, created_at, updated_at) VALUES ($1,$2,'API','api_'||$3,'sum','',NOW(),NOW())`,
		metricID, tenantID, run)
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New())
	subID := uuid.New()
	must(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at, progressive_billing_threshold)
		VALUES ($1,$2,$3,$4,'active',NOW(),NOW()+INTERVAL '1 month',NOW(),NOW(),NOW(),$5)`,
		subID, tenantID, custID, planID, int64(50000))
	chargeID := uuid.New()
	must(t, conn, `INSERT INTO plan_charges (id, tenant_id, plan_id, metric_id, charge_model, amounts, hsn_code, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'per_unit','{"INR":{"unit_amount":"1"}}',' ',NOW(),NOW())`,
		chargeID, tenantID, planID, metricID)
	return tenantID, subID, chargeID
}

func must(t *testing.T, conn *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
}

// TestProgressiveWatermarkCAS_NoDoubleCount proves the idempotency primitive on
// real Postgres: a retried advance that reads a stale oldAmount loses the CAS
// and does NOT re-bill; only monotonic advances win.
func TestProgressiveWatermarkCAS_NoDoubleCount(t *testing.T) {
	conn := openProgressiveTestDB(t)
	repo := NewProgressiveBillingRepository(conn)
	ctx := context.Background()
	tenantID, subID, chargeID := seedProgressiveScope(t, conn)
	period := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// GetThreshold reads the seeded 50000.
	th, err := repo.GetThreshold(ctx, subID)
	if err != nil || th == nil || *th != 50000 {
		t.Fatalf("threshold = %v (err %v), want 50000", th, err)
	}

	// First advance 0 -> 10000 creates the row and wins.
	won, err := repo.AdvanceWatermarkCAS(ctx, tenantID, subID, chargeID, period, 0, 10000)
	if err != nil || !won {
		t.Fatalf("first advance won=%v err=%v, want true", won, err)
	}
	if wm, _ := repo.GetWatermark(ctx, subID, chargeID, period); wm != 10000 {
		t.Fatalf("watermark = %d, want 10000", wm)
	}

	// A retry that still thinks old=0 must LOSE (row is already 10000) -> no
	// double-bill. This is the core guarantee.
	won, err = repo.AdvanceWatermarkCAS(ctx, tenantID, subID, chargeID, period, 0, 10000)
	if err != nil || won {
		t.Fatalf("stale retry won=%v err=%v, want false (no double-count)", won, err)
	}
	if wm, _ := repo.GetWatermark(ctx, subID, chargeID, period); wm != 10000 {
		t.Fatalf("watermark after stale retry = %d, want unchanged 10000", wm)
	}

	// A genuine advance 10000 -> 25000 wins.
	won, err = repo.AdvanceWatermarkCAS(ctx, tenantID, subID, chargeID, period, 10000, 25000)
	if err != nil || !won {
		t.Fatalf("second advance won=%v, want true", won)
	}

	// Two concurrent runs both read old=25000; exactly one may advance.
	winA, _ := repo.AdvanceWatermarkCAS(ctx, tenantID, subID, chargeID, period, 25000, 40000)
	winB, _ := repo.AdvanceWatermarkCAS(ctx, tenantID, subID, chargeID, period, 25000, 41000)
	if winA == winB {
		t.Fatalf("concurrent advances winA=%v winB=%v, want exactly one true", winA, winB)
	}
	if wm, _ := repo.GetWatermark(ctx, subID, chargeID, period); wm != 40000 && wm != 41000 {
		t.Fatalf("watermark after concurrent = %d, want 40000 or 41000 (one winner)", wm)
	}
}
