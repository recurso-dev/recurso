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

// TestDowngradeProration_CouponDiscountedPrices_Postgres proves R3 (ENG-195):
// plan-change proration uses the prices the customer actually pays THIS period,
// not list prices, when the current period's invoice carried the subscription's
// coupon discount.
//
// Shape: Pro 200000 with an 80%-off coupon — the customer paid 40000 for the
// period. Mid-period downgrade to Basic 100000.
//
//	Old (coupon-blind): credit = (200000 − 100000)/2 = ~50000 — MORE than the
//	40000 the customer paid for the WHOLE period. Spendable account credit
//	minted out of thin air (money-out over-credit); with a steeper discount or
//	an earlier downgrade the arbitrage grows without bound.
//	Fixed: discounted prices (Pro 40000, Basic 20000) → credit = ~10000, the
//	unused half of the discounted difference — what the customer actually paid
//	for the service being given up.
//
// A control subscription WITHOUT the discounted-period flag still prorates at
// list price, pinning that undiscounted periods are unchanged.
func TestDowngradeProration_CouponDiscountedPrices_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed coupon-proration test")
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
	proPlanID := seedPlan("Pro", 200000)
	basicPlanID := seedPlan("Basic", 100000)

	couponID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, active, created_at, updated_at)
		 VALUES ($1,$2,$3,'percent',80,'forever',TRUE,NOW(),NOW())`,
		couponID, tenantID, "EIGHTY-"+run); err != nil {
		t.Fatalf("seed coupon: %v", err)
	}

	// seedSub creates a mid-period Pro subscription with a paid, ledger-posted
	// invoice for `paid` and a matching recognition schedule. When discounted,
	// the sub carries the coupon and the discounted-period flag.
	seedSub := func(tag string, paid int64, discounted bool) uuid.UUID {
		customerID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
			 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
			customerID, tenantID, tag+"-"+run+"@t.com", uuid.New()); err != nil {
			t.Fatalf("seed customer %s: %v", tag, err)
		}
		subID := uuid.New()
		var cpID *uuid.UUID
		periods := 0
		if discounted {
			cpID = &couponID
			periods = 1
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor,
			                            coupon_id, coupon_periods_applied, coupon_applied_current_period, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days',
			         $5,$6,$7,NOW(),NOW())`,
			subID, tenantID, customerID, proPlanID, cpID, periods, discounted); err != nil {
			t.Fatalf("seed subscription %s: %v", tag, err)
		}
		invID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
			 VALUES ($1,$2,$3,$4,'USD',$6,$6,$6,'paid',$5,NOW(),NOW())`,
			invID, tenantID, customerID, subID, "INV-"+tag+"-"+run, paid); err != nil {
			t.Fatalf("seed invoice %s: %v", tag, err)
		}
		ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
		if err := ledger.RecordInvoice(ctx, &domain.Invoice{
			ID: invID, TenantID: tenantID, CustomerID: customerID, SubscriptionID: &subID,
			InvoiceNumber: "INV-" + tag + "-" + run, Total: paid, Currency: "USD",
		}); err != nil {
			t.Fatalf("RecordInvoice %s: %v", tag, err)
		}
		seedRevRecSchedule(t, conn, tenantID, invID, subID, paid/10, 10)
		return subID
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)
	svc := NewSubscriptionService(subRepo, db.NewInvoiceRepository(conn), db.NewPlanRepository(conn), db.NewCustomerRepository(dbx),
		db.NewCouponRepository(conn), nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	// Discounted sub: paid 40000 (80% off Pro). Downgrade credit must be ~10000
	// (half of the discounted 40000−20000 difference), never ~50000 list.
	discSub := seedSub("disc", 40000, true)
	if _, err := svc.UpdateSubscription(tctx, tenantID, discSub, basicPlanID); err != nil {
		t.Fatalf("UpdateSubscription (discounted downgrade): %v", err)
	}
	var discCredit int64
	if err := conn.QueryRowContext(ctx,
		`SELECT cn.amount FROM credit_notes cn JOIN subscriptions s ON s.customer_id = cn.customer_id
		  WHERE cn.tenant_id = $1 AND s.id = $2 AND cn.type = 'adjustment'`,
		tenantID, discSub).Scan(&discCredit); err != nil {
		t.Fatalf("read discounted credit note: %v", err)
	}
	if discCredit < 8000 || discCredit > 12000 {
		t.Errorf("discounted downgrade credit = %d, want ~10000 (discounted prices) — list-price proration mints credit the customer never paid", discCredit)
	}

	// Control sub: same shape, no discounted-period flag → list-price proration.
	ctrlSub := seedSub("ctrl", 200000, false)
	if _, err := svc.UpdateSubscription(tctx, tenantID, ctrlSub, basicPlanID); err != nil {
		t.Fatalf("UpdateSubscription (control downgrade): %v", err)
	}
	var ctrlCredit int64
	if err := conn.QueryRowContext(ctx,
		`SELECT cn.amount FROM credit_notes cn JOIN subscriptions s ON s.customer_id = cn.customer_id
		  WHERE cn.tenant_id = $1 AND s.id = $2 AND cn.type = 'adjustment'`,
		tenantID, ctrlSub).Scan(&ctrlCredit); err != nil {
		t.Fatalf("read control credit note: %v", err)
	}
	if ctrlCredit < 45000 || ctrlCredit > 55000 {
		t.Errorf("control downgrade credit = %d, want ~50000 (list prices unchanged for undiscounted periods)", ctrlCredit)
	}

	// Books stay clean end-to-end (the discounted credit fits inside the
	// discounted deferral, so nothing goes wrong-sign).
	tb, err := ledger.GetTrialBalance(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) carries an abnormal (wrong-sign) balance %d", l.Code, l.Name, l.Balance)
		}
	}
	if !tb.Balanced {
		t.Errorf("ledger does not balance: debits %d != credits %d", tb.TotalDebits, tb.TotalCredits)
	}
}
