package scheduler

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

const testAppURL = "https://app.test"

type capturingEmailSender struct {
	mu   sync.Mutex
	sent []port.EmailMessage
}

func (c *capturingEmailSender) Send(_ context.Context, m port.EmailMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, m)
	return nil
}

// to returns the one email sent to address, failing the test unless exactly one was.
func (c *capturingEmailSender) to(t *testing.T, address string) port.EmailMessage {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	var found []port.EmailMessage
	for _, m := range c.sent {
		if m.To == address {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("emails to %s = %d, want 1", address, len(found))
	}
	return found[0]
}

func openEmailLinksDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed email link test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// seedEmailCustomer creates a tenant and customer, with a saved card if asked.
func seedEmailCustomer(t *testing.T, conn *sql.DB, savedCard bool) (tenantID, customerID uuid.UUID, address string) {
	t.Helper()
	tenantID, customerID = uuid.New(), uuid.New()
	address = "links-" + customerID.String()[:8] + "@example.com"
	seed(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Links-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	seed(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "Links Customer", address, uuid.New())
	if savedCard {
		seed(t, conn, `UPDATE customers SET stripe_customer_id = 'cus_links', default_payment_method = 'pm_links', card_last4 = '4242' WHERE id = $1`, customerID)
	}
	return tenantID, customerID, address
}

func runDunningFor(t *testing.T, savedCard bool) (invoiceID uuid.UUID, msg port.EmailMessage) {
	t.Helper()
	conn := openEmailLinksDB(t)
	tenantID, customerID, address := seedEmailCustomer(t, conn, savedCard)
	invoiceID = uuid.New()
	seed(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied,
			status, invoice_number, retry_count, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',10000,10000,0,0,'open',$4,0, NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day')`,
		invoiceID, tenantID, customerID, "INV-LNK-"+invoiceID.String()[:8])

	sender := &capturingEmailSender{}
	notif := service.NewNotificationService(sender, "https://api.test")
	sched := NewDunningScheduler(db.NewInvoiceRepository(conn), notif, memory.NewNoOpLocker(), DefaultDunningConfig(), testAppURL)
	sched.processDunning()
	return invoiceID, sender.to(t, address)
}

func TestDunningFirstEmail_NoCard_DoesNotClaimAFailedCharge_Postgres(t *testing.T) {
	invoiceID, msg := runDunningFor(t, false)

	if strings.Contains(msg.Subject, "Failed") {
		t.Errorf("subject = %q, want no claim of a failed payment", msg.Subject)
	}
	for _, claim := range []string{"attempted to charge", "Payment Failed", "automatically retry", "Next Retry"} {
		if strings.Contains(msg.HTMLBody, claim) {
			t.Errorf("body claims %q with no card on file", claim)
		}
	}
	if want := testAppURL + "/checkout/" + invoiceID.String(); !strings.Contains(msg.HTMLBody, want) {
		t.Errorf("body has no Pay Now link to %s", want)
	}
}

func TestDunningFirstEmail_SavedCard_OffersPayNowAndCardUpdate_Postgres(t *testing.T) {
	invoiceID, msg := runDunningFor(t, true)

	if !strings.Contains(msg.Subject, "Payment Failed") {
		t.Errorf("subject = %q, want a failed payment", msg.Subject)
	}
	if !strings.Contains(msg.HTMLBody, "attempted to charge") {
		t.Error("body should say a charge was attempted")
	}
	for _, want := range []string{testAppURL + "/checkout/" + invoiceID.String(), testAppURL + "/portal/login"} {
		if !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("body has no link to %s", want)
		}
	}
}

func runPreChargeFor(t *testing.T, savedCard bool) port.EmailMessage {
	t.Helper()
	conn := openEmailLinksDB(t)
	tenantID, customerID, address := seedEmailCustomer(t, conn, savedCard)
	planID := uuid.New()
	seed(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Links Plan',$3,'month',1,true)`,
		planID, tenantID, "links-"+planID.String()[:8])
	seed(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type, created_at) VALUES ($1,$2,'USD',9900,'flat',NOW())`,
		uuid.New(), planID)
	seed(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 month', NOW() + INTERVAL '12 hours', NOW(), NOW(), NOW())`,
		uuid.New(), tenantID, customerID, planID)

	sender := &capturingEmailSender{}
	notif := service.NewNotificationService(sender, "https://api.test")
	subRepo := db.NewSubscriptionRepository(conn).(*db.SubscriptionRepository)
	sched := NewPreChargeScheduler(subRepo, notif, memory.NewNoOpLocker(), testAppURL)
	sched.runPreChargeNotifications()
	return sender.to(t, address)
}

func TestPreChargeReminder_SavedCard_ShowsCardAndWorkingLink_Postgres(t *testing.T) {
	msg := runPreChargeFor(t, true)

	if !strings.Contains(msg.HTMLBody, `href="`+testAppURL+`/portal/login"`) {
		t.Error("body has no link to the portal login route; bare /portal has no page")
	}
	if !strings.Contains(msg.HTMLBody, "•••• 4242") {
		t.Error("body does not name the card that will be charged")
	}
}

func TestPreChargeReminder_NoCard_DoesNotPromiseACharge_Postgres(t *testing.T) {
	msg := runPreChargeFor(t, false)

	for _, claim := range []string{"••••", "authorize us to charge", "No action is needed"} {
		if strings.Contains(msg.HTMLBody, claim) {
			t.Errorf("body says %q with no card on file", claim)
		}
	}
	if !strings.Contains(msg.HTMLBody, "invoice to pay") {
		t.Error("body should say an invoice is coming")
	}
}
