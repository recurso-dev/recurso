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

// TestUpgradeProration_PostsInvoiceLeg proves the F1 completeness invariant for
// mid-cycle upgrades: the proration CHARGE invoice must post its invoice leg
// (DR AR / CR Deferred) to the ledger, not just get settled later. Without it,
// the reconciler reports a missing_invoice_transaction and the ledger can never
// trial-balance for that invoice.
//
// The reconciliation service is the oracle: after a real upgrade, it must find
// ZERO missing_invoice_transaction discrepancies and the ledger must be balanced.
func TestUpgradeProration_PostsInvoiceLeg(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed upgrade-invoice-leg test")
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
	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	// US customer (no tax) keeps the arithmetic about completeness, not tax.
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return id
	}
	// UPGRADE: current cheap (100000) -> target pricey (200000).
	currentPlanID := seedPlan("Basic", 100000)
	targetPlanID := seedPlan("Pro", 200000)

	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Paid current-plan invoice, posted to the ledger so the baseline is clean
	// (only the upgrade's proration invoice is under test).
	curInvID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,'USD',100000,100000,100000,'paid',$5,NOW(),NOW())`,
		curInvID, tenantID, customerID, subID, "INV-"+run); err != nil {
		t.Fatalf("seed current invoice: %v", err)
	}
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	if err := ledger.RecordInvoice(ctx, &domain.Invoice{
		ID: curInvID, TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-" + run, Total: 100000, Currency: "USD",
	}); err != nil {
		t.Fatalf("RecordInvoice (baseline): %v", err)
	}
	// Also settle the baseline invoice's cash leg so the baseline reconciles clean.
	if err := ledger.RecordPayment(ctx, &domain.Invoice{
		ID: curInvID, TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: "INV-" + run, Total: 100000, Currency: "USD",
	}); err != nil {
		t.Fatalf("RecordPayment (baseline): %v", err)
	}
	seedRevRecSchedule(t, conn, tenantID, curInvID, subID, 10000, 10) // pending 100000

	// Wire the subscription service with the real ledger + rev-rec and drive the
	// upgrade. This creates the proration CHARGE invoice.
	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)
	svc := NewSubscriptionService(subRepo, db.NewInvoiceRepository(conn), db.NewPlanRepository(conn),
		db.NewCustomerRepository(dbx), nil, nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (upgrade): %v", err)
	}

	// The upgrade must have created a proration charge invoice.
	var prorationInvoices int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM invoices WHERE subscription_id = $1 AND id <> $2 AND status <> 'draft'`,
		subID, curInvID).Scan(&prorationInvoices); err != nil {
		t.Fatalf("count proration invoices: %v", err)
	}
	if prorationInvoices == 0 {
		t.Fatal("expected the upgrade to create a proration charge invoice")
	}

	// ORACLE: reconciliation must find no missing invoice leg and a balanced ledger.
	recon := NewReconciliationService(db.NewLedgerRepository(conn), nil)
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("reconciliation Run: %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyMissingInvoiceTx {
			t.Errorf("proration charge invoice missing its ledger invoice leg (F1): %+v", d)
		}
		if d.Type == DiscrepancyLedgerUnbalanced || d.Type == DiscrepancyAbnormalBalance {
			t.Errorf("ledger not audit-grade after upgrade: %s %+v", d.Type, d)
		}
	}
}
