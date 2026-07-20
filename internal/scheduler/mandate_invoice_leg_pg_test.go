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

// TestMandateDebit_PostsInvoiceLeg proves the F1 completeness invariant for
// mandate debits: the debit invoice must post its invoice leg (DR AR / CR
// Deferred, plus the GST reclassification) to the ledger at charge time, not
// only get a cash leg when the webhook settles it. The reconciliation service
// is the oracle — after a real scheduler debit it must find ZERO
// missing_invoice_transaction discrepancies and a balanced ledger.
func TestMandateDebit_PostsInvoiceLeg(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed mandate invoice-leg test")
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
		tenantID, "MD-LEG-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	custRepo := db.NewCustomerRepository(sqlxConn)
	name := "Mandate Customer"
	customer := &domain.Customer{
		ID: uuid.New(), TenantID: tenantID, Name: &name,
		Email: "md-" + tenantID.String()[:8] + "@example.com", Phone: "+919999999999",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := custRepo.Create(tenantCtx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "md-pro-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type, created_at) VALUES ($1,$2,'INR',12000,'recurring',NOW())`,
		uuid.New(), planID)
	subID := uuid.New()
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW(), NOW(), NOW(), NOW())`,
		subID, tenantID, customer.ID, planID)

	mandateID := uuid.New()
	seed(t, conn, `INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
			razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'upi','upi','md@upi','tok_md','','',50000,'monthly','active',TRUE, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		mandateID, tenantID, customer.ID, subID)

	mandateRepo := db.NewMandateRepository(conn)
	ledger := service.NewLedgerService(nil, db.NewLedgerRepository(conn))
	mandateSvc := service.NewMandateService(mandateRepo, &gateway.MockGateway{}, custRepo, db.NewInvoiceRepository(conn))
	mandateSvc.SetBillingResolver(db.NewSubscriptionRepository(conn), db.NewPlanRepository(conn), fixedGSTResolver{})
	mandateSvc.SetLedgerService(ledger) // F1: post the debit invoice's ledger leg

	sched := NewMandateDebitScheduler(mandateRepo, mandateSvc, memory.NewNoOpLocker())
	sched.runDebits()

	// One debit invoice must exist...
	var invCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM invoices WHERE customer_id = $1`, customer.ID).Scan(&invCount); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if invCount != 1 {
		t.Fatalf("invoices created = %d, want 1", invCount)
	}

	// ...and it must carry its ledger invoice leg (reconciliation is the oracle).
	recon := service.NewReconciliationService(db.NewLedgerRepository(conn), nil)
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("reconciliation Run: %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.Type == service.DiscrepancyMissingInvoiceTx {
			t.Errorf("mandate debit invoice missing its ledger invoice leg (F1): %+v", d)
		}
		if d.Type == service.DiscrepancyLedgerUnbalanced || d.Type == service.DiscrepancyAbnormalBalance {
			t.Errorf("ledger not audit-grade after mandate debit: %s %+v", d.Type, d)
		}
	}
}
