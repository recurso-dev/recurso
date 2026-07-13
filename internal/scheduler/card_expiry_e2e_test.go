package scheduler

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/service"
)

// TestCardExpiringScheduler_RunNotifications_Postgres drives the card-expiry
// scheduler loop against a real DB: a customer with an active subscription whose
// card expires next month gets a card-expiry notification recorded (so it isn't
// re-sent). Regression guard for the card-expiry notification loop.
func TestCardExpiringScheduler_RunNotifications_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed card-expiry e2e")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	sqlxConn := sqlx.NewDb(conn, "postgres")
	ctx := context.Background()

	next := time.Now().AddDate(0, 1, 0)
	month, year := int(next.Month()), next.Year()

	tenantID := uuid.New()
	seed(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "CE-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	seed(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, card_brand, card_last4, card_exp_month, card_exp_year, created_at)
		VALUES ($1,$2,$3,$4,$5,'visa','4242',$6,$7,NOW())`,
		customerID, tenantID, "Card Cust", "ce-"+customerID.String()[:8]+"@example.com", uuid.New(), month, year)
	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "ce-pro-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW(), NOW() + INTERVAL '1 month', NOW(), NOW(), NOW())`,
		uuid.New(), tenantID, customerID, planID)

	custRepo := db.NewCustomerRepository(sqlxConn) // *db.CustomerRepository satisfies CustomerRepoForCardExpiry
	notif := service.NewNotificationService(noopEmailSender{}, "http://example.test")
	sched := NewCardExpiringScheduler(custRepo, notif, memory.NewNoOpLocker(), "http://portal.test")

	sched.runCardExpiryNotifications()

	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM card_expiry_notifications WHERE customer_id = $1 AND card_exp_month = $2 AND card_exp_year = $3`,
		customerID, month, year).Scan(&count); err != nil {
		t.Fatalf("count card expiry notifications: %v", err)
	}
	if count != 1 {
		t.Fatalf("card expiry notifications for customer = %d, want 1", count)
	}
}
