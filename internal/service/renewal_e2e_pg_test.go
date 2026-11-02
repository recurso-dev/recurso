package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// tenantClaims runs the real claim query but hands back only this test's
// rows, so a sweep on a shared database never renews another test's
// subscriptions.
type tenantClaims struct {
	*db.SubscriptionRepository
	tenantID uuid.UUID
}

func (c tenantClaims) ClaimDueForRenewal(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	subs, err := c.SubscriptionRepository.ClaimDueForRenewal(ctx, lease, limit)
	if err != nil {
		return nil, err
	}
	var own []*domain.Subscription
	for _, s := range subs {
		if s.TenantID == c.tenantID {
			own = append(own, s)
		}
	}
	return own, nil
}

// scriptedCharger records saved-card charges and declines the listed customers.
type scriptedCharger struct {
	declined map[string]bool
	charges  []string // idempotency keys, in order
	amounts  []int64
}

func (c *scriptedCharger) ChargeSavedPaymentMethod(_ context.Context, stripeCustomerID, _ string, amount int64, _, _, idempotencyKey string) (*port.PaymentResult, error) {
	c.charges = append(c.charges, idempotencyKey)
	c.amounts = append(c.amounts, amount)
	return &port.PaymentResult{Success: !c.declined[stripeCustomerID]}, nil
}

// renewalE2E wires the renewal engine as cmd/api does (real repositories,
// invoice service, ledger, rev-rec and settlement) with only the gateway faked.
type renewalE2E struct {
	h       *invariantHarness
	svc     *RenewalService
	charger *scriptedCharger
}

func newRenewalE2E(t *testing.T, conn *sql.DB, dbx *sqlx.DB) *renewalE2E {
	t.Helper()
	h := newInvariantHarness(t, conn, dbx)

	subRepo := db.NewSubscriptionRepository(conn)
	planRepo := db.NewPlanRepository(conn)
	customerRepo := db.NewCustomerRepository(dbx)
	invoices := NewInvoiceService(db.NewInvoiceRepository(conn), planRepo, customerRepo,
		db.NewUnbilledChargeRepository(conn), subRepo, nil, NewTaxResolver(harnessGSTConfigs{}, "United States", ""))
	invoices.CouponRepo = db.NewCouponRepository(conn)
	invoices.AddonRepo = db.NewSubscriptionAddonRepository(conn)
	invoices.LedgerPoster = h.ledger
	invoices.RevRecScheduler = h.revrec

	charger := &scriptedCharger{declined: map[string]bool{}}
	svc := NewRenewalService(tenantClaims{subRepo.(*db.SubscriptionRepository), h.tenantID}, planRepo, invoices)
	svc.SetSavedMethodCharging(charger, customerRepo, h.subSvc)
	return &renewalE2E{h: h, svc: svc, charger: charger}
}

// seedDue inserts a customer and an active subscription whose period ended
// `overdue` ago. A non-empty card saves a payment method on the customer.
func (e *renewalE2E) seedDue(overdue string, cancelAtPeriodEnd bool, card string) (subID, customerID uuid.UUID) {
	h := e.h
	h.t.Helper()
	h.n++
	customerID, subID = uuid.New(), uuid.New()
	var stripeID, method any
	if card != "" {
		stripeID, method = card, "pm_"+card
	}
	mustExec(h.t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, stripe_customer_id, default_payment_method, created_at, updated_at)
		VALUES ($1,$2,$3,'Renewal Cust','United States','individual',$4,$5,$6,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("renew-%s-%d@t.com", h.run, h.n), uuid.New(), stripeID, method)
	mustExec(h.t, h.conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, cancel_at_period_end, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month' - $5::interval, NOW() - $5::interval, NOW() - INTERVAL '1 month' - $5::interval, $6, NOW(), NOW())`,
		subID, h.tenantID, customerID, h.cheapPlan, overdue, cancelAtPeriodEnd)
	return subID, customerID
}

// sweep calls ProcessDueRenewals exactly as BillingCycleScheduler does: on
// context.Background(), with no tenant. Leases are cleared first so a due row
// is re-claimed the way it would be once its lease lapses.
func (e *renewalE2E) sweep() {
	h := e.h
	h.t.Helper()
	mustExec(h.t, h.conn, `UPDATE subscriptions SET renewal_claimed_at = NULL WHERE tenant_id = $1`, h.tenantID)
	if _, err := e.svc.ProcessDueRenewals(context.Background()); err != nil {
		h.t.Fatalf("ProcessDueRenewals: %v", err)
	}
}

type renewalInvoiceRow struct {
	status string
	total  int64
	paid   int64
}

func (e *renewalE2E) invoices(subID uuid.UUID) []renewalInvoiceRow {
	h := e.h
	h.t.Helper()
	rows, err := h.conn.QueryContext(h.ctx,
		`SELECT status, total, amount_paid FROM invoices WHERE subscription_id = $1 ORDER BY created_at`, subID)
	if err != nil {
		h.t.Fatalf("list invoices: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []renewalInvoiceRow
	for rows.Next() {
		var r renewalInvoiceRow
		if err := rows.Scan(&r.status, &r.total, &r.paid); err != nil {
			h.t.Fatalf("scan invoice: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("iterate invoices: %v", err)
	}
	return out
}

func (e *renewalE2E) subState(subID uuid.UUID) (status string, due bool) {
	h := e.h
	h.t.Helper()
	if err := h.conn.QueryRowContext(h.ctx,
		`SELECT status, current_period_end <= NOW() FROM subscriptions WHERE id = $1`, subID).Scan(&status, &due); err != nil {
		h.t.Fatalf("read subscription: %v", err)
	}
	return status, due
}

// TestRenewalSweep_EndToEnd_Postgres drives scheduled renewals through the real
// repositories, invoice service, ledger and settlement. It exists because the
// sweep's tenant-less context once failed every scheduled renewal, which unit
// tests on fakes could not see: the failure lives in the tenant-scoped plan and
// invoice lookups. Every outcome must leave the ledger audit-grade.
func TestRenewalSweep_EndToEnd_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed renewal e2e test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	dbx := sqlx.NewDb(conn, "postgres")

	const planAmount = int64(100000)
	e := newRenewalE2E(t, conn, dbx)

	paidSub, _ := e.seedDue("1 minute", false, "cus_pays")
	openSub, _ := e.seedDue("1 minute", false, "")
	declinedSub, _ := e.seedDue("1 minute", false, "cus_declines")
	e.charger.declined["cus_declines"] = true
	cancelSub, _ := e.seedDue("1 minute", true, "cus_cancels")
	behindSub, _ := e.seedDue("1 month 1 minute", false, "")

	e.sweep()
	e.h.assertAuditGrade("after first sweep")

	t.Run("saved card is charged and the invoice settles", func(t *testing.T) {
		got := e.invoices(paidSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].total != planAmount || got[0].paid != planAmount {
			t.Fatalf("invoices = %+v, want one paid invoice of %d", got, planAmount)
		}
		if status, due := e.subState(paidSub); status != "active" || due {
			t.Errorf("subscription status=%s due=%v, want active and no longer due", status, due)
		}
	})

	t.Run("no saved card leaves the invoice open", func(t *testing.T) {
		got := e.invoices(openSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if _, due := e.subState(openSub); due {
			t.Error("period did not advance")
		}
	})

	t.Run("declined card leaves the invoice open for dunning", func(t *testing.T) {
		got := e.invoices(declinedSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if _, due := e.subState(declinedSub); due {
			t.Error("period did not advance")
		}
	})

	t.Run("cancel at period end cancels without a flat fee or charge", func(t *testing.T) {
		for _, inv := range e.invoices(cancelSub) {
			if inv.total >= planAmount {
				t.Errorf("final invoice %+v carries the flat fee for a period that will not run", inv)
			}
		}
		if status, _ := e.subState(cancelSub); status != string(domain.SubscriptionStatusCanceled) {
			t.Errorf("status = %s, want canceled", status)
		}
	})

	t.Run("a subscription behind by a cycle catches up one invoice per sweep", func(t *testing.T) {
		if got := e.invoices(behindSub); len(got) != 1 {
			t.Fatalf("after first sweep: %d invoices, want 1", len(got))
		}
		if _, due := e.subState(behindSub); !due {
			t.Fatal("after first sweep: want still due")
		}
	})

	if len(e.charger.charges) != 2 {
		t.Errorf("charges = %v, want exactly the paying and declining customers", e.charger.charges)
	}

	// The next sweep may only bill the subscription still behind; everything
	// else is undue, so a repeat must neither re-invoice nor re-charge.
	e.sweep()
	e.h.assertAuditGrade("after second sweep")

	if got := e.invoices(behindSub); len(got) != 2 {
		t.Errorf("behind subscription: %d invoices after catch-up, want 2", len(got))
	}
	if _, due := e.subState(behindSub); due {
		t.Error("behind subscription: still due after catch-up")
	}
	for name, id := range map[string]uuid.UUID{"paid": paidSub, "open": openSub, "declined": declinedSub} {
		if got := e.invoices(id); len(got) != 1 {
			t.Errorf("%s subscription re-invoiced on repeat sweep: %+v", name, got)
		}
	}
	if len(e.charger.charges) != 2 {
		t.Errorf("repeat sweep charged again: %v", e.charger.charges)
	}

	e.h.assertClosePackTies("after renewals")
}
