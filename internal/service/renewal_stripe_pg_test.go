package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	stripeclient "github.com/stripe/stripe-go/v76/client"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/gateway"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// recordingCharger passes charges through to the real gateway and keeps what
// came back, so assertions can read the resulting PaymentIntent from Stripe.
type recordingCharger struct {
	inner   renewalSavedCharger
	results map[string][]*port.PaymentResult
}

func (c *recordingCharger) ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error) {
	res, err := c.inner.ChargeSavedPaymentMethod(ctx, stripeCustomerID, paymentMethodID, amount, currency, invoiceID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	c.results[stripeCustomerID] = append(c.results[stripeCustomerID], res)
	return res, nil
}

// settlerFailingOnce fails its first settlement, the crash window between a
// successful charge and marking the invoice paid.
type settlerFailingOnce struct {
	renewalSettler
	failed bool
}

func (s *settlerFailingOnce) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (bool, error) {
	if !s.failed {
		s.failed = true
		return false, errors.New("injected settlement failure")
	}
	return s.renewalSettler.MarkInvoicePaid(ctx, invoiceID)
}

// TestRenewalSweep_RealStripe_Postgres runs scheduled renewals against Stripe's
// test mode, so charges, declines and idempotent replays are Stripe's own
// behaviour rather than a fake's. It needs STRIPE_TEST_SECRET_KEY and refuses
// anything but a test key.
func TestRenewalSweep_RealStripe_Postgres(t *testing.T) {
	key := os.Getenv("STRIPE_TEST_SECRET_KEY")
	if key == "" {
		t.Skip("STRIPE_TEST_SECRET_KEY not set; skipping real Stripe renewal test")
	}
	if !strings.HasPrefix(key, "sk_test_") {
		t.Fatal("STRIPE_TEST_SECRET_KEY must be a test-mode key (sk_test_)")
	}
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h

	sc := stripeclient.New(key, nil)
	charger := &recordingCharger{inner: gateway.NewStripeGateway(key, ""), results: map[string][]*port.PaymentResult{}}
	settler := &settlerFailingOnce{renewalSettler: h.subSvc}
	customerRepo := db.NewCustomerRepository(dbx)
	e.svc.SetSavedMethodCharging(charger, customerRepo, h.subSvc)
	e.svc.SetPaymentIDRecorder(db.NewInvoiceRepository(conn))

	// seedCard saves one of Stripe's test payment methods on a real test customer.
	seedCard := func(testMethod string) (subID uuid.UUID, stripeCustomerID string) {
		t.Helper()
		cust, err := sc.Customers.New(&stripe.CustomerParams{Description: stripe.String("recurso renewal test " + h.run)})
		if err != nil {
			t.Fatalf("create stripe customer: %v", err)
		}
		t.Cleanup(func() { _, _ = sc.Customers.Del(cust.ID, nil) })
		pm, err := sc.PaymentMethods.Attach(testMethod, &stripe.PaymentMethodAttachParams{Customer: stripe.String(cust.ID)})
		if err != nil {
			t.Fatalf("attach %s: %v", testMethod, err)
		}
		subID, customerID := e.seedDue("1 minute", false, cust.ID)
		mustExec(t, conn, `UPDATE customers SET default_payment_method = $1 WHERE id = $2`, pm.ID, customerID)
		return subID, cust.ID
	}
	onlyResult := func(t *testing.T, stripeCustomerID string) *port.PaymentResult {
		t.Helper()
		got := charger.results[stripeCustomerID]
		if len(got) != 1 {
			t.Fatalf("gateway results = %+v, want one", got)
		}
		return got[0]
	}

	paysSub, paysCust := seedCard("pm_card_visa")
	declinedSub, declinedCust := seedCard("pm_card_chargeCustomerFail")
	authSub, authCust := seedCard("pm_card_authenticationRequired")

	e.sweep()
	h.assertAuditGrade("after real Stripe sweep")

	t.Run("a good card is really charged and the invoice settles", func(t *testing.T) {
		got := e.invoices(paysSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) || got[0].paid != got[0].total {
			t.Fatalf("invoices = %+v, want one paid invoice", got)
		}
		res := onlyResult(t, paysCust)
		pi, err := sc.PaymentIntents.Get(res.PaymentID, nil)
		if err != nil {
			t.Fatalf("read payment intent: %v", err)
		}
		if pi.Status != stripe.PaymentIntentStatusSucceeded || pi.AmountReceived != got[0].total {
			t.Errorf("payment intent status=%s received=%d, want succeeded for %d", pi.Status, pi.AmountReceived, got[0].total)
		}
		if pi.Currency != "usd" || pi.Customer == nil || pi.Customer.ID != paysCust {
			t.Errorf("payment intent currency=%s customer=%+v, want usd on %s", pi.Currency, pi.Customer, paysCust)
		}
		if pi.Metadata["invoice_id"] == "" {
			t.Error("payment intent has no invoice_id, so its webhook can't find the invoice")
		}
		var recorded string
		if err := conn.QueryRowContext(h.ctx, `SELECT COALESCE(gateway_payment_id, '') FROM invoices WHERE subscription_id = $1`, paysSub).Scan(&recorded); err != nil {
			t.Fatalf("read gateway payment id: %v", err)
		}
		if recorded != res.PaymentID {
			t.Errorf("gateway payment id = %q, want %q so a refund can reach Stripe", recorded, res.PaymentID)
		}
	})

	t.Run("a declined card leaves the invoice open with Stripe's decline code", func(t *testing.T) {
		got := e.invoices(declinedSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if res := onlyResult(t, declinedCust); res.Success || res.ErrorCode != "card_declined" {
			t.Errorf("result = %+v, want a card_declined failure", res)
		}
	})

	t.Run("a card needing 3DS can't be charged off-session and stays open", func(t *testing.T) {
		got := e.invoices(authSub)
		if len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) || got[0].paid != 0 {
			t.Fatalf("invoices = %+v, want one open unpaid invoice", got)
		}
		if res := onlyResult(t, authCust); res.Success || res.ErrorCode != "authentication_required" {
			t.Errorf("result = %+v, want an authentication_required failure", res)
		}
	})

	t.Run("a crash after the charge can't charge the card twice", func(t *testing.T) {
		e.svc.SetSavedMethodCharging(charger, customerRepo, settler)
		crashSub, crashCust := seedCard("pm_card_visa")
		e.sweep()
		h.assertAuditGrade("after settlement failure")

		var invoiceID uuid.UUID
		if err := conn.QueryRowContext(h.ctx, `SELECT id FROM invoices WHERE subscription_id = $1`, crashSub).Scan(&invoiceID); err != nil {
			t.Fatalf("read invoice: %v", err)
		}
		inv, err := db.NewInvoiceRepository(conn).GetByID(h.tctx, invoiceID)
		if err != nil {
			t.Fatalf("load invoice: %v", err)
		}
		if inv.Status != domain.InvoiceStatusOpen {
			t.Fatalf("status = %s after the injected failure, want open", inv.Status)
		}
		e.svc.AttemptPayment(h.tctx, inv)

		got := charger.results[crashCust]
		if len(got) != 2 || got[0].PaymentID == "" || got[0].PaymentID != got[1].PaymentID {
			t.Fatalf("results = %+v, want the same payment intent replayed", got)
		}
		iter := sc.PaymentIntents.List(&stripe.PaymentIntentListParams{Customer: stripe.String(crashCust)})
		count := 0
		for iter.Next() {
			count++
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("list payment intents: %v", err)
		}
		if count != 1 {
			t.Errorf("stripe holds %d payment intents for the customer, want 1", count)
		}
		if rows := e.invoices(crashSub); len(rows) != 1 || rows[0].status != string(domain.InvoiceStatusPaid) {
			t.Errorf("invoices = %+v, want one paid invoice", rows)
		}
	})

	h.assertAuditGrade("after real Stripe renewals")
}
