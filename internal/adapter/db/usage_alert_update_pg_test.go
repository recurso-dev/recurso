package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestUsageAlertUpdateThreshold_Postgres proves the alert-edit SQL: the
// threshold updates and the per-period fired dedup resets; a missing/foreign
// alert is sql.ErrNoRows; a duplicate (subscription, metric, type, threshold)
// surfaces the unique violation.
func TestUsageAlertUpdateThreshold_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed usage-alert update test")
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

	tenantID, custID := seedCreditAppTenantCustomer(t, conn)
	repo := NewUsageAlertRepository(conn)

	// usage_alerts.subscription_id is FK'd — seed a real plan + subscription.
	run := uuid.New().String()[:8]
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "alertpro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW(), NOW() + INTERVAL '1 month', NOW(), NOW(), NOW())`,
		subID, tenantID, custID, planID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	mkAlert := func(threshold int64) *domain.UsageAlert {
		a := &domain.UsageAlert{
			ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, MetricCode: "api_calls",
			ThresholdType: domain.AlertThresholdQuantity, Threshold: threshold,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		if err := repo.Create(ctx, a); err != nil {
			t.Fatalf("create alert: %v", err)
		}
		return a
	}

	a := mkAlert(1000)
	b := mkAlert(2000) // sibling with the threshold we'll collide into

	// Simulate a firing, then edit — the dedup must reset.
	if ok, err := repo.MarkFired(ctx, a.ID, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)); err != nil || !ok {
		t.Fatalf("MarkFired: ok=%v err=%v", ok, err)
	}
	if err := repo.UpdateThreshold(ctx, tenantID, a.ID, domain.AlertThresholdQuantity, 5000); err != nil {
		t.Fatalf("UpdateThreshold: %v", err)
	}
	var threshold int64
	var lastFired *time.Time
	if err := conn.QueryRowContext(ctx,
		`SELECT threshold, last_fired_period_start FROM usage_alerts WHERE id=$1`, a.ID).
		Scan(&threshold, &lastFired); err != nil {
		t.Fatalf("read alert: %v", err)
	}
	if threshold != 5000 {
		t.Errorf("threshold = %d, want 5000", threshold)
	}
	if lastFired != nil {
		t.Error("editing the threshold must reset last_fired_period_start")
	}

	// Colliding with the sibling's (sub, metric, type, threshold) → unique violation.
	if err := repo.UpdateThreshold(ctx, tenantID, a.ID, domain.AlertThresholdQuantity, b.Threshold); err == nil || !IsUniqueViolation(err) {
		t.Errorf("duplicate threshold: err = %v, want a unique violation", err)
	}

	// Unknown id and cross-tenant → sql.ErrNoRows.
	if err := repo.UpdateThreshold(ctx, tenantID, uuid.New(), domain.AlertThresholdQuantity, 7000); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("unknown alert: err = %v, want sql.ErrNoRows", err)
	}
	if err := repo.UpdateThreshold(ctx, uuid.New(), a.ID, domain.AlertThresholdQuantity, 7000); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("cross-tenant edit: err = %v, want sql.ErrNoRows", err)
	}
}
