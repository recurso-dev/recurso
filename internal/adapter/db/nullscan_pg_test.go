package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBackgroundJobNullScans_Postgres locks in the ENG-143 fixes: the
// cross-tenant background sweeps (nexus/churn tenant iteration, the pre-charge
// notifier, and the dunning retry sweep) must survive the NULLs and
// timestamptz columns they actually meet in a real database. Each of these
// used to abort the entire sweep on the first offending row.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database, e.g.:
//
//	createdb recurso_repo_test
//	TEST_DATABASE_URL='postgres://localhost:5432/recurso_repo_test?sslmode=disable' go test ./internal/adapter/db/
func TestBackgroundJobNullScans_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed null-scan test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	// Unique key material so repeat runs against a long-lived scratch DB don't
	// collide on the tenants/customers/plans unique constraints.
	run := uuid.New().String()[:8]

	// --- ListTenants: a tenant with a NULL email (the live failure — 1 of the
	// dev tenants has no email, and the nexus + churn schedulers iterate every
	// tenant). ---
	nullEmailTenant := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at)
		 VALUES ($1, $2, NULL, NOW(), NOW())`,
		nullEmailTenant, "NullEmail-"+run); err != nil {
		t.Fatalf("seed null-email tenant: %v", err)
	}

	tenantRepo := NewTenantRepository(conn)
	tenants, err := tenantRepo.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants must not abort on a NULL email: %v", err)
	}
	found := false
	for _, tn := range tenants {
		if tn.ID == nullEmailTenant {
			found = true
			if tn.Email != "" {
				t.Errorf("NULL email should scan to empty string, got %q", tn.Email)
			}
		}
	}
	if !found {
		t.Fatal("ListTenants did not return the NULL-email tenant")
	}

	// --- GetSubscriptionsDueTomorrow + MarkPreChargeNotificationSent: a
	// subscription renewing in ~12h whose customer has a NULL name. Exercises
	// the timestamptz->time scan and the nullable customer name, then the
	// prices JOIN (the old query read plans.price/plans.currency, columns that
	// do not exist, so it errored on every call). ---
	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "PreCharge-"+run, "billing-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at)
		 VALUES ($1, $2, $3, NULL, $4, NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer (NULL name): %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active)
		 VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO prices (id, plan_id, currency, amount, type)
		 VALUES ($1, $2, 'INR', 199900, 'recurring')`,
		uuid.New(), planID); err != nil {
		t.Fatalf("seed price: %v", err)
	}
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status,
			current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'active', NOW(), NOW() + INTERVAL '12 hours', NOW(), NOW(), NOW())`,
		subID, tenantID, customerID, planID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Concrete type: these background-job methods aren't on the port interface.
	subRepo := &SubscriptionRepository{db: conn}
	due, err := subRepo.GetSubscriptionsDueTomorrow(ctx)
	if err != nil {
		t.Fatalf("GetSubscriptionsDueTomorrow (timestamptz + NULL name): %v", err)
	}
	var got bool
	for _, s := range due {
		if s.ID == subID {
			got = true
			if s.NextBillingDate == "" {
				t.Error("NextBillingDate should be a formatted date, got empty")
			}
			if s.CustomerName != "" {
				t.Errorf("NULL customer name should scan to empty, got %q", s.CustomerName)
			}
			if s.Amount != 199900 || s.Currency != "INR" {
				t.Errorf("price via JOIN prices = %d %s, want 199900 INR", s.Amount, s.Currency)
			}
		}
	}
	if !got {
		t.Fatal("subscription due in 12h was not returned")
	}

	// The prices JOIN must resolve — the old plans.price query errored here.
	if err := subRepo.MarkPreChargeNotificationSent(ctx, subID, time.Now().Format("2006-01-02")); err != nil {
		t.Fatalf("MarkPreChargeNotificationSent (prices JOIN): %v", err)
	}

	// --- GetDueForRetry: a worker-managed retry-eligible invoice whose
	// e-invoice columns are all NULL (a non-e-invoiced row — the common case).
	// The raw string scan used to abort the whole retry sweep. ---
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, subscription_id, customer_id, currency,
			subtotal, total, status, invoice_number, next_retry_at, dunning_managed_by,
			hsn_code, irn, ack_no, signed_qr_code, e_invoice_status)
		 VALUES ($1, $2, $3, $4, 'INR', 199900, 199900, 'open', $5,
			NOW() - INTERVAL '1 hour', 'worker',
			NULL, NULL, NULL, NULL, NULL)`,
		invID, tenantID, subID, customerID, "INV-"+run); err != nil {
		t.Fatalf("seed retry invoice (NULL e-invoice cols): %v", err)
	}

	invRepo := NewInvoiceRepository(conn)
	retry, err := invRepo.GetDueForRetry(ctx)
	if err != nil {
		t.Fatalf("GetDueForRetry must not abort on NULL e-invoice columns: %v", err)
	}
	var retried bool
	for _, inv := range retry {
		if inv.ID == invID {
			retried = true
			if inv.HSNCode != "" || inv.IRN != "" || inv.EInvoiceStatus != "" {
				t.Errorf("NULL e-invoice cols should scan to empty, got hsn=%q irn=%q status=%q",
					inv.HSNCode, inv.IRN, inv.EInvoiceStatus)
			}
		}
	}
	if !retried {
		t.Fatal("worker-managed retry invoice was not returned")
	}

	// --- ClaimDueForRetry (ADR-003): the atomic claim must return the same due
	// row, lease it by advancing next_retry_at into the future, and a second
	// immediate claim must NOT return it (a concurrent instance is locked out
	// until the lease lapses). ---
	claimed, err := invRepo.ClaimDueForRetry(ctx, 10*time.Minute, 10)
	if err != nil {
		t.Fatalf("ClaimDueForRetry: %v", err)
	}
	var claimedFound bool
	for _, inv := range claimed {
		if inv.ID == invID {
			claimedFound = true
			if inv.NextRetryAt == nil || !inv.NextRetryAt.After(time.Now()) {
				t.Errorf("claim must lease next_retry_at into the future, got %v", inv.NextRetryAt)
			}
		}
	}
	if !claimedFound {
		t.Fatal("ClaimDueForRetry did not return the due retry invoice")
	}
	// A second claim in the same window must skip the just-leased row.
	again, err := invRepo.ClaimDueForRetry(ctx, 10*time.Minute, 10)
	if err != nil {
		t.Fatalf("second ClaimDueForRetry: %v", err)
	}
	for _, inv := range again {
		if inv.ID == invID {
			t.Error("a leased invoice must not be re-claimed within its lease window")
		}
	}
}
