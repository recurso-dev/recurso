package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestPlanDowngradeCredit_Postgres proves the ENG-150 fix end-to-end: a plan
// downgrade persists its proration credit as a spendable adjustment CREDIT NOTE
// (not a force-zeroed $0 "paid" invoice that silently vanished), and flips the
// subscription to the new plan. Runs against the real schema/repos.
func TestPlanDowngradeCredit_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed downgrade-credit test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Downgrade-"+run, "downgrade-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	// US customer → no GST/VAT, so the credit equals the pre-tax net exactly.
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// Two plans: current ($2000/mo) and cheaper target ($1000/mo). Mid-period the
	// change nets a credit for the unused portion of the pricier plan.
	seedPlan := func(name string, amountMinor int64) uuid.UUID {
		planID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active)
			 VALUES ($1, $2, $3, $4, 'month', 1, TRUE)`,
			planID, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1, $2, 'USD', $3, 'recurring')`,
			uuid.New(), planID, amountMinor); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return planID
	}
	currentPlanID := seedPlan("Pro", 200000)
	targetPlanID := seedPlan("Basic", 100000)

	// Active subscription mid-period (started 15d ago, ends in 15d).
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Wire the service with real repos; credit-note repo injected via the setter.
	subRepo := db.NewSubscriptionRepository(conn)
	invoiceRepo := db.NewInvoiceRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	svc := NewSubscriptionService(
		subRepo, invoiceRepo, planRepo, customerRepo,
		nil, nil, nil, nil, nil,
		db.NewTxManager(conn), nil, nil,
	)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	updated, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID)
	if err != nil {
		t.Fatalf("UpdateSubscription (downgrade): %v", err)
	}

	// The subscription must now point at the cheaper plan.
	if updated.PlanID != targetPlanID {
		t.Fatalf("subscription plan_id = %s, want target %s", updated.PlanID, targetPlanID)
	}

	// A spendable adjustment credit note must exist for the customer.
	var (
		cnAmount, cnBalance int64
		cnType, cnStatus    string
		cnReason, cnRefund  string
		cnCurrency          string
	)
	err = conn.QueryRowContext(ctx,
		`SELECT amount, balance, type, status, reason, refund_status, currency
		 FROM credit_notes WHERE tenant_id = $1 AND customer_id = $2`,
		tenantID, customerID).Scan(&cnAmount, &cnBalance, &cnType, &cnStatus, &cnReason, &cnRefund, &cnCurrency)
	if err != nil {
		t.Fatalf("expected one downgrade credit note, got: %v", err)
	}
	if cnType != string(domain.CreditNoteTypeAdjustment) {
		t.Errorf("credit note type = %q, want adjustment", cnType)
	}
	if cnStatus != string(domain.CreditNoteStatusIssued) {
		t.Errorf("credit note status = %q, want issued", cnStatus)
	}
	if cnRefund != string(domain.RefundStatusNone) {
		t.Errorf("credit note refund_status = %q, want none", cnRefund)
	}
	if cnCurrency != "USD" {
		t.Errorf("credit note currency = %q, want USD", cnCurrency)
	}
	if cnAmount <= 0 || cnBalance != cnAmount {
		t.Errorf("credit note amount/balance = %d/%d, want positive & equal (spendable)", cnAmount, cnBalance)
	}
	// Mid-period 2000→1000 downgrade nets roughly a $500 credit ($1000 unused on
	// the old plan minus $500 charge on the new). Assert the ballpark, not an
	// exact figure (proration depends on the exact NOW()).
	if cnAmount < 40000 || cnAmount > 60000 {
		t.Errorf("credit note amount = %d minor units, want ~50000 ($500)", cnAmount)
	}

	// The old $0 "paid" invoice must NOT exist: no invoice should have been
	// written for this downgrade (ENG-150 — the credit is a credit note now).
	var invCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM invoices WHERE subscription_id = $1`, subID).Scan(&invCount); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if invCount != 0 {
		t.Errorf("downgrade created %d invoice(s); want 0 (credit is a credit note)", invCount)
	}
}

// TestPlanUpgradeCharge_Postgres proves the ENG-150 atomic charge path: an
// upgrade writes the proration CHARGE invoice and flips the plan in a single
// transaction (SubscriptionRepository.UpdateWithTx + InvoiceRepository.CreateWithTx),
// producing exactly one invoice and no credit note.
func TestPlanUpgradeCharge_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed upgrade-charge test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Upgrade-"+run, "upgrade-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amountMinor int64) uuid.UUID {
		planID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active)
			 VALUES ($1, $2, $3, $4, 'month', 1, TRUE)`,
			planID, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1, $2, 'USD', $3, 'recurring')`,
			uuid.New(), planID, amountMinor); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return planID
	}
	currentPlanID := seedPlan("Starter", 100000)
	targetPlanID := seedPlan("Pro", 200000)

	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	subRepo := db.NewSubscriptionRepository(conn)
	invoiceRepo := db.NewInvoiceRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	svc := NewSubscriptionService(
		subRepo, invoiceRepo, planRepo, customerRepo,
		nil, nil, nil, nil, nil,
		db.NewTxManager(conn), nil, nil,
	)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	updated, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID)
	if err != nil {
		t.Fatalf("UpdateSubscription (upgrade): %v", err)
	}
	if updated.PlanID != targetPlanID {
		t.Fatalf("subscription plan_id = %s, want target %s (atomic flip)", updated.PlanID, targetPlanID)
	}

	// Exactly one proration charge invoice, tax-inclusive total > 0.
	var invCount int
	var invTotal, invSubtotal int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total),0), COALESCE(SUM(subtotal),0) FROM invoices WHERE subscription_id = $1`,
		subID).Scan(&invCount, &invTotal, &invSubtotal); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if invCount != 1 {
		t.Fatalf("upgrade created %d invoice(s); want exactly 1", invCount)
	}
	if invSubtotal <= 0 || invTotal <= 0 {
		t.Errorf("upgrade invoice subtotal/total = %d/%d, want positive charge", invSubtotal, invTotal)
	}

	// The DB reflects the committed plan change (atomic with the invoice write).
	var dbPlanID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT plan_id FROM subscriptions WHERE id = $1`, subID).Scan(&dbPlanID); err != nil {
		t.Fatalf("read subscription plan_id: %v", err)
	}
	if dbPlanID != targetPlanID {
		t.Errorf("persisted plan_id = %s, want %s", dbPlanID, targetPlanID)
	}

	// No credit note for an upgrade.
	var cnCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_notes WHERE tenant_id = $1 AND customer_id = $2`,
		tenantID, customerID).Scan(&cnCount); err != nil {
		t.Fatalf("count credit notes: %v", err)
	}
	if cnCount != 0 {
		t.Errorf("upgrade created %d credit note(s); want 0", cnCount)
	}
}
