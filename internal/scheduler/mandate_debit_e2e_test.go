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
	"github.com/recurso-dev/recurso/internal/adapter/gateway"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// TestMandateDebitScheduler_RunDebits_Postgres drives the mandate-debit
// scheduler's core loop against a real DB: a due mandate must be charged (one
// invoice created) and its schedule advanced to the next full cycle. This is
// the regression guard the founder's rule asks for — and it catches the
// tenant-context class: the loop runs with a background context, so ExecuteDebit
// must inject the mandate's own tenant before the tenant-scoped customer read.
func TestMandateDebitScheduler_RunDebits_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed mandate-debit e2e")
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

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "MD-E2E-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	custRepo := db.NewCustomerRepository(sqlxConn)
	name := "Mandate Customer"
	customer := &domain.Customer{
		ID: uuid.New(), TenantID: tenantID, Name: &name,
		Email:     "md-" + tenantID.String()[:8] + "@example.com",
		Phone:     "+919999999999",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := custRepo.Create(tenantCtx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	// A due mandate: active, pre-notified, next_debit_at already in the past.
	mandateID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO mandates (id, tenant_id, customer_id, mandate_type, payment_method, vpa,
			razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
		 VALUES ($1,$2,$3,'upi','upi','md@upi','tok_md','','',50000,'monthly','active',TRUE, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		mandateID, tenantID, customer.ID); err != nil {
		t.Fatalf("seed mandate: %v", err)
	}

	mandateRepo := db.NewMandateRepository(conn)
	invRepo := db.NewInvoiceRepository(conn)
	mandateSvc := service.NewMandateService(mandateRepo, &gateway.MockGateway{}, custRepo, invRepo)
	sched := NewMandateDebitScheduler(mandateRepo, mandateSvc, memory.NewNoOpLocker())

	sched.runDebits()

	// The due mandate must have been charged — one invoice for the customer.
	var invCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM invoices WHERE customer_id = $1`, customer.ID).Scan(&invCount); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if invCount != 1 {
		t.Fatalf("invoices created = %d, want 1 (scheduler failed to charge the due mandate)", invCount)
	}

	// The mandate advanced to the next full cycle (~1 month), not the short claim
	// window, and the pre-notify flag reset.
	var nextDebit time.Time
	var preNotified bool
	if err := conn.QueryRowContext(ctx,
		`SELECT next_debit_at, pre_debit_notified FROM mandates WHERE id = $1`, mandateID).
		Scan(&nextDebit, &preNotified); err != nil {
		t.Fatalf("read mandate: %v", err)
	}
	if !nextDebit.After(time.Now().Add(20 * 24 * time.Hour)) {
		t.Errorf("next_debit_at = %v, want ~1 month out (full-cycle advance after a successful debit)", nextDebit)
	}
	if preNotified {
		t.Error("pre_debit_notified should be reset to false after a debit")
	}
}
