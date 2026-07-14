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
	"github.com/recurso-dev/recurso/internal/core/port"
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

	// The subscription the mandate bills: a plan with a ₹120.00 recurring price.
	// The mandate's authorization ceiling (max_amount) is 50000 — far above the
	// real charge — so a debit of the ceiling (the old bug) is unmistakable.
	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "md-pro-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type, created_at) VALUES ($1,$2,'INR',12000,'recurring',NOW())`,
		uuid.New(), planID)
	subID := uuid.New()
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW(), NOW(), NOW(), NOW())`,
		subID, tenantID, customer.ID, planID)

	// A due mandate linked to that subscription: active, pre-notified,
	// next_debit_at already in the past.
	mandateID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
			razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'upi','upi','md@upi','tok_md','','',50000,'monthly','active',TRUE, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		mandateID, tenantID, customer.ID, subID); err != nil {
		t.Fatalf("seed mandate: %v", err)
	}

	mandateRepo := db.NewMandateRepository(conn)
	invRepo := db.NewInvoiceRepository(conn)
	mandateSvc := service.NewMandateService(mandateRepo, &gateway.MockGateway{}, custRepo, invRepo)
	// Charge the subscription's real recurring amount (plan price + tax), capped
	// at the mandate ceiling (ENG-165). A fixed 18% GST resolver keeps the tax
	// deterministic so the asserted total is exact.
	mandateSvc.SetBillingResolver(db.NewSubscriptionRepository(conn), db.NewPlanRepository(conn), fixedGSTResolver{})
	sched := NewMandateDebitScheduler(mandateRepo, mandateSvc, memory.NewNoOpLocker())

	sched.runDebits()

	// The due mandate must have been charged — exactly one invoice — and for the
	// subscription's real amount (₹120 plan price + 18% GST = 14160), NOT the
	// 50000 authorization ceiling that the old code debited (ENG-165). The GST
	// split must be stamped on the invoice, not left flat.
	var invCount int
	var subtotal, taxAmount, cgst, sgst, total int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) OVER (), subtotal, tax_amount, cgst_amount, sgst_amount, total
		   FROM invoices WHERE customer_id = $1`, customer.ID).
		Scan(&invCount, &subtotal, &taxAmount, &cgst, &sgst, &total); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	if invCount != 1 {
		t.Fatalf("invoices created = %d, want 1 (scheduler failed to charge the due mandate)", invCount)
	}
	if total == 50000 {
		t.Fatal("charged 50000 — the mandate MaxAmount ceiling (ENG-165 regression)")
	}
	if subtotal != 12000 || taxAmount != 2160 || cgst != 1080 || sgst != 1080 || total != 14160 {
		t.Errorf("invoice = subtotal %d / tax %d (cgst %d, sgst %d) / total %d, want 12000 / 2160 (1080,1080) / 14160 (plan price + 18%% GST)",
			subtotal, taxAmount, cgst, sgst, total)
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

// countingGateway counts mandate debits so a test can assert at-most-once.
type countingGateway struct {
	*gateway.MockGateway
	debits int
}

func (g *countingGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	g.debits++
	return &port.PaymentResult{Success: true, PaymentID: "pay_e164"}, nil
}

// TestMandateDebit_SameCycleChargesAtMostOnce is the ENG-164 guard: because
// Razorpay does not honor the idempotency header, a re-attempt of the same
// billing cycle (e.g. after a crash that lost the schedule advance) must be
// stopped locally. The OPEN invoice's per-cycle UNIQUE key is that guard — the
// second attempt's invoice insert conflicts, so it skips the charge instead of
// double-charging.
func TestMandateDebit_SameCycleChargesAtMostOnce(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed at-most-once test")
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
	seed(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "MD-once-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	custRepo := db.NewCustomerRepository(sqlxConn)
	name := "Once Cust"
	customer := &domain.Customer{
		ID: uuid.New(), TenantID: tenantID, Name: &name,
		Email:     "once-" + tenantID.String()[:8] + "@example.com",
		Phone:     "+919999999999",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := custRepo.Create(tenantCtx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro',$3,'month',1,TRUE)`,
		planID, tenantID, "once-pro-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type, created_at) VALUES ($1,$2,'INR',12000,'recurring',NOW())`,
		uuid.New(), planID)
	subID := uuid.New()
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW(), NOW(), NOW(), NOW())`,
		subID, tenantID, customer.ID, planID)

	// First-ever cycle: last_debit_at NULL -> cycle key "md-<id>-0".
	mandateID := uuid.New()
	seed(t, conn, `INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa,
		razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'upi','upi','md@upi','tok_md','','',50000,'monthly','active',TRUE, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		mandateID, tenantID, customer.ID, subID)

	mandateRepo := db.NewMandateRepository(conn)
	invRepo := db.NewInvoiceRepository(conn)
	gw := &countingGateway{MockGateway: &gateway.MockGateway{}}
	svc := service.NewMandateService(mandateRepo, gw, custRepo, invRepo)
	svc.SetBillingResolver(db.NewSubscriptionRepository(conn), db.NewPlanRepository(conn), fixedGSTResolver{})

	load := func() *domain.Mandate {
		m, err := mandateRepo.GetByID(ctx, mandateID, tenantID)
		if err != nil {
			t.Fatalf("load mandate: %v", err)
		}
		return m
	}

	// First debit: claims the cycle, advances the schedule, charges once.
	if err := svc.DebitSubscription(ctx, load()); err != nil {
		t.Fatalf("first debit: %v", err)
	}

	// Simulate the schedule advance being LOST (crash after charging): reset to the
	// same cycle (last_debit_at NULL) and make it due again.
	seed(t, conn, `UPDATE mandates SET last_debit_at = NULL, next_debit_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, mandateID)

	// Re-attempt the SAME cycle: the invoice claim must conflict and skip the charge.
	if err := svc.DebitSubscription(ctx, load()); err != nil {
		t.Fatalf("second debit (same cycle): %v", err)
	}

	if gw.debits != 1 {
		t.Errorf("gateway charged %d times, want exactly 1 (a re-attempt of the same cycle must not double-charge)", gw.debits)
	}
	var invCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM invoices WHERE customer_id = $1`, customer.ID).Scan(&invCount); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if invCount != 1 {
		t.Errorf("invoices = %d, want 1 (the per-cycle claim prevents a second invoice)", invCount)
	}
	// The conflict path still advances the schedule so it stops re-claiming.
	if m := load(); m.NextDebitAt == nil || !m.NextDebitAt.After(time.Now().Add(20*24*time.Hour)) {
		t.Error("next_debit_at not advanced ~1 month out after the conflict-path advance")
	}
}

// fixedGSTResolver is a deterministic 18% intra-state GST resolver for the test,
// so the charged total is exact. It satisfies the mandate service's tax-resolver
// dependency without pulling in the real jurisdiction engine.
type fixedGSTResolver struct{}

func (fixedGSTResolver) ResolveInvoiceTax(_ context.Context, _ uuid.UUID, _ *domain.Customer, _ string, amount int64, hsn string) service.InvoiceTax {
	tax := amount * 18 / 100
	half := tax / 2
	if hsn == "" {
		hsn = "998314"
	}
	return service.InvoiceTax{Total: tax, CGST: half, SGST: tax - half, TaxType: "intra_state", Rate: 0.18, HSN: hsn}
}
