package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// capturedEmails records what would have been sent. Invoice emails from the
// sweep are asynchronous, so readers wait for them.
type capturedEmails struct {
	mu   sync.Mutex
	sent []port.EmailMessage
}

func (c *capturedEmails) Send(_ context.Context, msg port.EmailMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, msg)
	return nil
}

func (c *capturedEmails) to(address string) []port.EmailMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []port.EmailMessage
	for _, m := range c.sent {
		if m.To == address {
			out = append(out, m)
		}
	}
	return out
}

func invoiceEmailsIn(msgs []port.EmailMessage) []port.EmailMessage {
	var out []port.EmailMessage
	for _, m := range msgs {
		if strings.Contains(m.HTMLBody, "/checkout/") {
			out = append(out, m)
		}
	}
	return out
}

func (c *capturedEmails) waitForInvoiceEmail(t *testing.T, address string) port.EmailMessage {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if got := invoiceEmailsIn(c.to(address)); len(got) > 0 {
			return got[0]
		}
	}
	t.Fatalf("no invoice email reached %s", address)
	return port.EmailMessage{}
}

func (c *capturedEmails) assertNoInvoiceEmail(t *testing.T, address string) {
	t.Helper()
	time.Sleep(300 * time.Millisecond) // long enough for an async send to land
	if got := invoiceEmailsIn(c.to(address)); len(got) != 0 {
		t.Errorf("%s was sent %d invoice emails, want none", address, len(got))
	}
}

// TestInvoiceEmails_Postgres proves a customer who has to pay by hand is told
// how, on real repositories, and that a customer about to be charged is not
// invited to pay twice. The sweep's tenant-less context once silently stopped
// every automatic invoice email, which fakes could not show.
func TestInvoiceEmails_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h
	emails := &capturedEmails{}
	notifications := NewNotificationService(emails, "https://app.test")
	e.invoice.NotificationService = notifications
	h.subSvc.SetNotificationService(notifications)
	h.subSvc.SetFinalUsageInvoicer(e.invoice)
	h.subSvc.SetFinalInvoiceCharger(e.svc)

	addressOf := func(customerID uuid.UUID) (address string) {
		t.Helper()
		if err := conn.QueryRowContext(h.ctx, `SELECT email FROM customers WHERE id = $1`, customerID).Scan(&address); err != nil {
			t.Fatalf("read customer email: %v", err)
		}
		return address
	}
	invoiceOf := func(subID uuid.UUID) (id string) {
		t.Helper()
		if err := conn.QueryRowContext(h.ctx, `SELECT id FROM invoices WHERE subscription_id = $1 ORDER BY created_at DESC LIMIT 1`, subID).Scan(&id); err != nil {
			t.Fatalf("read invoice: %v", err)
		}
		return id
	}

	noCardSub, noCard := e.seedDue("1 minute", false, "")
	cardSub, card := e.seedDue("1 minute", false, "cus_email_pays")
	declinedSub, declined := e.seedDue("1 minute", false, "cus_email_declines")
	e.charger.declined["cus_email_declines"] = true
	e.sweep()

	t.Run("a renewal with no saved card emails the payment link", func(t *testing.T) {
		msg := emails.waitForInvoiceEmail(t, addressOf(noCard))
		if want := "https://app.test/checkout/" + invoiceOf(noCardSub); !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("email has no link to %s", want)
		}
	})
	t.Run("a renewal charged to a saved card sends no payment link", func(t *testing.T) {
		emails.assertNoInvoiceEmail(t, addressOf(card))
		if got := e.invoices(cardSub); len(got) != 1 || got[0].status != string(domain.InvoiceStatusPaid) {
			t.Errorf("invoices = %+v, want one paid", got)
		}
	})
	t.Run("a declined renewal leaves the payment link to dunning", func(t *testing.T) {
		emails.assertNoInvoiceEmail(t, addressOf(declined))
		if got := e.invoices(declinedSub); len(got) != 1 || got[0].status != string(domain.InvoiceStatusOpen) {
			t.Errorf("invoices = %+v, want one open", got)
		}
	})

	t.Run("a new subscription emails its first invoice and doesn't claim it is paid for", func(t *testing.T) {
		_, customerID := e.seedDue("-15 days", false, "")
		address := addressOf(customerID)
		sub, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan,
		})
		if err != nil {
			t.Fatalf("create subscription: %v", err)
		}
		msg := emails.waitForInvoiceEmail(t, address)
		if want := "/checkout/" + invoiceOf(sub.ID); !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("email has no link to %s", want)
		}
		for _, m := range emails.to(address) {
			if strings.Contains(m.HTMLBody, "activated successfully") {
				t.Error("an unpaid subscription was told it is activated")
			}
		}
	})

	t.Run("a free subscription gets no payment link", func(t *testing.T) {
		free := e.seedCoupon("percent", 100, "forever", 0)
		var code string
		if err := conn.QueryRowContext(h.ctx, `SELECT code FROM coupons WHERE id = $1`, free).Scan(&code); err != nil {
			t.Fatalf("read coupon: %v", err)
		}
		_, customerID := e.seedDue("-15 days", false, "")
		address := addressOf(customerID)
		if _, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan, CouponCode: code,
		}); err != nil {
			t.Fatalf("create subscription: %v", err)
		}
		emails.assertNoInvoiceEmail(t, address)
		welcome := emails.to(address)
		if len(welcome) != 1 || !strings.Contains(welcome[0].HTMLBody, "activated successfully") {
			t.Errorf("emails = %d, want one activation email", len(welcome))
		}
	})

	t.Run("a free renewal gets no payment link", func(t *testing.T) {
		free := e.seedCoupon("percent", 100, "forever", 0)
		subID, customerID := e.seedDue("1 minute", false, "")
		e.attachCoupon(subID, free, 1)
		e.sweep()
		emails.assertNoInvoiceEmail(t, addressOf(customerID))
	})

	t.Run("a trial's first invoice emails the payment link", func(t *testing.T) {
		_, customerID := e.seedDue("-15 days", false, "")
		sub, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan, TrialDays: 7,
		})
		if err != nil {
			t.Fatalf("create trial: %v", err)
		}
		address := addressOf(customerID)
		emails.assertNoInvoiceEmail(t, address)
		if _, err := h.subSvc.ConvertTrialToActive(h.tctx, sub); err != nil {
			t.Fatalf("convert trial: %v", err)
		}
		emails.waitForInvoiceEmail(t, address)
	})

	t.Run("a free trial's first invoice gets no payment link", func(t *testing.T) {
		free := e.seedCoupon("percent", 100, "forever", 0)
		var code string
		if err := conn.QueryRowContext(h.ctx, `SELECT code FROM coupons WHERE id = $1`, free).Scan(&code); err != nil {
			t.Fatalf("read coupon: %v", err)
		}
		_, customerID := e.seedDue("-15 days", false, "")
		sub, err := h.subSvc.CreateSubscription(h.tctx, CreateSubscriptionInput{
			TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan, CouponCode: code, TrialDays: 7,
		})
		if err != nil {
			t.Fatalf("create trial: %v", err)
		}
		if _, err := h.subSvc.ConvertTrialToActive(h.tctx, sub); err != nil {
			t.Fatalf("convert trial: %v", err)
		}
		emails.assertNoInvoiceEmail(t, addressOf(customerID))
	})

	t.Run("a final usage invoice follows the saved card too", func(t *testing.T) {
		const dimension = "api_calls"
		planID := e.seedMeteredPlan(dimension)
		usage := func(subID, customerID uuid.UUID) {
			t.Helper()
			if err := db.NewUsageRepository(conn).RecordEvent(h.ctx, &domain.UsageEvent{
				ID: uuid.New(), SubscriptionID: subID, CustomerID: customerID,
				Dimension: dimension, Quantity: 40, Timestamp: time.Now().Add(-time.Hour),
			}); err != nil {
				t.Fatalf("record usage: %v", err)
			}
		}
		noCardFinal, noCardFinalCustomer := e.seedDueOnPlan(planID, "-15 days", false, "")
		usage(noCardFinal, noCardFinalCustomer)
		cardFinal, cardFinalCustomer := e.seedDueOnPlan(planID, "-15 days", false, "cus_email_final")
		usage(cardFinal, cardFinalCustomer)
		for _, subID := range []uuid.UUID{noCardFinal, cardFinal} {
			if _, err := h.subSvc.Cancel(h.tctx, h.tenantID, subID, true, "", ""); err != nil {
				t.Fatalf("cancel: %v", err)
			}
		}
		emails.waitForInvoiceEmail(t, addressOf(noCardFinalCustomer))
		emails.assertNoInvoiceEmail(t, addressOf(cardFinalCustomer))
	})
}
