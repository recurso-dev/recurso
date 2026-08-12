package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestPaymentAttempt_Lifecycle_Postgres exercises the async settlement lifecycle:
// create → in-flight → settle, keyed on the PaymentIntent id the webhook uses.
func TestPaymentAttempt_Lifecycle_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payment-attempt test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	database, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := database.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "PA-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID) }()

	// Per-run suffix: the payment-intent id hits a unique index
	// (idx_payment_attempts_pi), so a hardcoded value fails on any REUSED
	// test database even though the tenant row is cleaned up (the ON DELETE
	// path doesn't cover payment_attempts' unique key from a crashed run).
	run := tenantID.String()[:8]
	piID := "pi_test_ach_" + run

	invoiceID := uuid.New()
	if _, err := database.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, status, created_at)
		 VALUES ($1,$2,'USD',100000,108750,$3,'open',NOW())`,
		invoiceID, tenantID, "INV-PA-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}

	repo := NewPaymentAttemptRepository(database)

	// Create a processing ACH attempt.
	att := &domain.PaymentAttempt{
		TenantID:               tenantID,
		InvoiceID:              invoiceID,
		Gateway:                "stripe",
		Method:                 "us_bank_account",
		GatewayPaymentIntentID: piID,
		Status:                 domain.PaymentAttemptProcessing,
		Amount:                 108750,
	}
	if err := repo.Create(ctx, att); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Webhook resolves it by PaymentIntent id.
	got, err := repo.GetByPaymentIntentID(ctx, piID)
	if err != nil || got == nil {
		t.Fatalf("get by pi: %v (got=%v)", err, got)
	}
	if got.Status != domain.PaymentAttemptProcessing || !got.InFlight() {
		t.Fatalf("expected in-flight processing, got %+v", got)
	}

	// Dunning-guard: the invoice has an in-flight attempt.
	if inflight, err := repo.HasInFlightForInvoice(ctx, invoiceID); err != nil || !inflight {
		t.Fatalf("HasInFlightForInvoice = %v, %v; want true", inflight, err)
	}

	// Settle it (payment_intent.succeeded).
	now := time.Now().UTC()
	if err := repo.UpdateStatusByPaymentIntent(ctx, piID, domain.PaymentAttemptSucceeded, "", &now); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.GetByPaymentIntentID(ctx, piID)
	if got.Status != domain.PaymentAttemptSucceeded || got.InFlight() {
		t.Fatalf("expected settled succeeded, got %+v", got)
	}
	if got.SettledAt == nil {
		t.Error("settled_at not set on success")
	}
	if inflight, _ := repo.HasInFlightForInvoice(ctx, invoiceID); inflight {
		t.Error("no attempt should be in-flight after settlement")
	}

	// Updating an unknown intent errors (not a silent no-op).
	if err := repo.UpdateStatusByPaymentIntent(ctx, "pi_missing", domain.PaymentAttemptFailed, "R01", nil); err == nil {
		t.Error("expected error updating a nonexistent attempt")
	}
}
