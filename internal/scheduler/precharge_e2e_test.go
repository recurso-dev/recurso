package scheduler

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/service"
)

// TestPreChargeScheduler_RunNotifications_Postgres drives the pre-charge
// scheduler's loop against a real DB: a subscription renewing within the
// notification window gets a pre-charge notification recorded (so it isn't
// re-notified). Regression guard for the RBI 24h pre-charge notice loop.
func TestPreChargeScheduler_RunNotifications_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed pre-charge e2e")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	seed(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "PC-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	seed(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "PreCharge Cust", "pc-"+customerID.String()[:8]+"@example.com", uuid.New())
	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "pc-pro-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type, created_at) VALUES ($1,$2,'INR',50000,'recurring',NOW())`,
		uuid.New(), planID)
	subID := uuid.New()
	// Renews ~12h from now — inside the 25h pre-charge window.
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW() + INTERVAL '12 hours', NOW(), NOW(), NOW())`,
		subID, tenantID, customerID, planID)

	subRepoPort := db.NewSubscriptionRepository(conn)
	subRepo := subRepoPort.(interface {
		GetSubscriptionsDueTomorrow(ctx context.Context) ([]db.SubscriptionWithCustomer, error)
		MarkPreChargeNotificationSent(ctx context.Context, subscriptionID uuid.UUID, chargeDate string) error
	})
	notif := service.NewNotificationService(noopEmailSender{}, "http://example.test")
	sched := NewPreChargeScheduler(subRepo, notif, memory.NewNoOpLocker(), "http://portal.test")

	sched.runPreChargeNotifications()

	// A notification row was recorded for this subscription's charge date, so it
	// will be excluded from the next run (no duplicate).
	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM precharge_notifications WHERE subscription_id = $1`, subID).Scan(&count); err != nil {
		t.Fatalf("count precharge notifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("precharge notifications for subscription = %d, want 1", count)
	}
}

func seed(t *testing.T, conn *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
}
