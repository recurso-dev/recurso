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

// openUsageAggTestDB connects to the CI Postgres (skips locally) and returns a
// UsageRepository plus the raw conn for seeding.
func openUsageAggTestDB(t *testing.T) (*UsageRepository, *sql.DB) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed usage aggregation test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewUsageRepository(conn), conn
}

// seedAggSubscription creates the tenant/plan/customer/subscription chain that
// usage_events foreign-keys onto, and returns the subscription + customer ids.
func seedAggSubscription(t *testing.T, conn *sql.DB) (subID, custID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Agg-"+run, "agg-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	custID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	subID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW(), NOW() + INTERVAL '1 month', NOW(), NOW(), NOW())`,
		subID, tenantID, custID, planID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	return subID, custID
}

// seedAggEvent inserts one usage event at ts with the given quantity.
func seedAggEvent(t *testing.T, conn *sql.DB, subID, custID uuid.UUID, dimension string, quantity int64, ts time.Time) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), subID, custID, dimension, quantity, ts); err != nil {
		t.Fatalf("seed event: %v", err)
	}
}

func TestAggregateForMetric_LatestAndPercentile(t *testing.T) {
	repo, conn := openUsageAggTestDB(t)
	ctx := context.Background()

	subID, custID := seedAggSubscription(t, conn)
	dim := "latency_ms_" + uuid.NewString()[:8]
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	// 100 events with quantities 1..100 within the period, in time order.
	for i := int64(1); i <= 100; i++ {
		seedAggEvent(t, conn, subID, custID, dim, i, start.Add(time.Duration(i)*time.Minute))
	}
	// An out-of-period event that must be ignored (later timestamp, huge value).
	seedAggEvent(t, conn, subID, custID, dim, 99999, end.Add(time.Hour))

	// latest: the most recent in-period event is quantity 100 (t = start+100m).
	latest := domain.BillableMetric{Code: dim, AggregationType: domain.AggregationLatest}
	got, err := repo.AggregateForMetric(ctx, subID, latest, start, end)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if got != 100 {
		t.Fatalf("latest = %d, want 100", got)
	}

	// percentile p95 of 1..100 is ~95 (percentile_cont interpolates to 95.05 -> 95).
	p95 := domain.BillableMetric{Code: dim, AggregationType: domain.AggregationPercentile, FieldName: "95"}
	got, err = repo.AggregateForMetric(ctx, subID, p95, start, end)
	if err != nil {
		t.Fatalf("p95: %v", err)
	}
	if got < 94 || got > 96 {
		t.Fatalf("p95 = %d, want ~95", got)
	}

	// p99 ~ 99.
	p99 := domain.BillableMetric{Code: dim, AggregationType: domain.AggregationPercentile, FieldName: "99"}
	got, err = repo.AggregateForMetric(ctx, subID, p99, start, end)
	if err != nil {
		t.Fatalf("p99: %v", err)
	}
	if got < 98 || got > 100 {
		t.Fatalf("p99 = %d, want ~99", got)
	}
}

func TestAggregateForMetric_LatestAndPercentileEmpty(t *testing.T) {
	repo, _ := openUsageAggTestDB(t)
	ctx := context.Background()
	subID := uuid.New()
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	dim := "empty_" + uuid.NewString()[:8]

	for _, agg := range []domain.AggregationType{domain.AggregationLatest, domain.AggregationPercentile} {
		m := domain.BillableMetric{Code: dim, AggregationType: agg, FieldName: "95"}
		got, err := repo.AggregateForMetric(ctx, subID, m, start, end)
		if err != nil {
			t.Fatalf("%s empty: %v", agg, err)
		}
		if got != 0 {
			t.Fatalf("%s empty = %d, want 0", agg, got)
		}
	}
}
