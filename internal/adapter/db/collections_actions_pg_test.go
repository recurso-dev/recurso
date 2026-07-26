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

// TestCollectionsActions_Postgres proves the Inc 3 manual controls at the SQL
// level: requeue only fires for a tenant-owned past_due non-mandate un-paused
// invoice, pausing excludes a row from the retry claim, and the manual write-off
// only flips a still-collectible invoice.
func TestCollectionsActions_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed collections-actions test")
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
	run := uuid.New().String()[:8]
	repo := NewInvoiceRepository(conn)

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Act-"+run, "act-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		custID, tenantID, "actc-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// seedInvoice returns the id; mandateKey empty means a normal (retryable) invoice.
	seedInvoice := func(status string, total int64, mandateKey string) uuid.UUID {
		id := uuid.New()
		var mk interface{}
		if mandateKey != "" {
			mk = mandateKey
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, mandate_cycle_key, dunning_managed_by, next_retry_at, created_at, due_date)
			 VALUES ($1, $2, $3, 'USD', $4, $4, 0, 0, $5, $6, $7, 'scheduler', NULL, NOW(), NOW() - INTERVAL '5 days')`,
			id, tenantID, custID, total, status, "INV-"+id.String()[:8], mk); err != nil {
			t.Fatalf("seed invoice (%s): %v", status, err)
		}
		return id
	}

	// --- RequeueForRetry: past_due non-mandate → requeued to the worker. ---
	pd := seedInvoice("past_due", 10000, "")
	ok, err := repo.RequeueForRetry(ctx, tenantID, pd)
	if err != nil || !ok {
		t.Fatalf("RequeueForRetry past_due: ok=%v err=%v", ok, err)
	}
	var managedBy string
	var nextRetry *time.Time
	if err := conn.QueryRowContext(ctx, `SELECT dunning_managed_by, next_retry_at FROM invoices WHERE id=$1`, pd).
		Scan(&managedBy, &nextRetry); err != nil {
		t.Fatalf("read requeued: %v", err)
	}
	if managedBy != "worker" || nextRetry == nil {
		t.Errorf("requeue must hand to worker with next_retry set: managed_by=%q next_retry=%v", managedBy, nextRetry)
	}

	// It should now be claimable by the retry worker.
	claimed, err := repo.ClaimDueForRetry(ctx, time.Minute, 10)
	if err != nil {
		t.Fatalf("ClaimDueForRetry: %v", err)
	}
	if !containsInvoice(claimed, pd) {
		t.Error("requeued invoice should be claimable by the worker")
	}

	// --- Mandate invoice can never be manually requeued. ---
	mand := seedInvoice("past_due", 5000, "md-"+run+"-1")
	if ok, _ := repo.RequeueForRetry(ctx, tenantID, mand); ok {
		t.Error("a mandate invoice must not be requeued (double-charge safety)")
	}

	// --- Pause excludes a row from the retry claim. ---
	paused := seedInvoice("past_due", 7000, "")
	if _, err := repo.RequeueForRetry(ctx, tenantID, paused); err != nil {
		t.Fatalf("requeue paused-candidate: %v", err)
	}
	if ok, err := repo.SetDunningPaused(ctx, tenantID, paused, true); err != nil || !ok {
		t.Fatalf("SetDunningPaused: ok=%v err=%v", ok, err)
	}
	claimed2, err := repo.ClaimDueForRetry(ctx, time.Minute, 10)
	if err != nil {
		t.Fatalf("ClaimDueForRetry after pause: %v", err)
	}
	if containsInvoice(claimed2, paused) {
		t.Error("a paused invoice must NOT be claimed by the retry worker")
	}

	// A paused invoice also can't be manually requeued.
	if ok, _ := repo.RequeueForRetry(ctx, tenantID, paused); ok {
		t.Error("a paused invoice must not be requeued")
	}

	// --- Eligibility reflects state. ---
	elig, err := repo.GetRetryEligibility(ctx, tenantID, paused)
	if err != nil {
		t.Fatalf("GetRetryEligibility: %v", err)
	}
	if !elig.Found || !elig.Paused || elig.IsMandate {
		t.Errorf("eligibility(paused) = %+v, want found+paused, not mandate", elig)
	}
	if e, _ := repo.GetRetryEligibility(ctx, tenantID, mand); !e.IsMandate {
		t.Errorf("eligibility(mandate).IsMandate = false, want true")
	}
	if e, _ := repo.GetRetryEligibility(ctx, tenantID, uuid.New()); e.Found {
		t.Error("eligibility for a missing invoice must be Found=false")
	}

	// --- Manual write-off only flips a collectible invoice. ---
	wo := seedInvoice("past_due", 3000, "")
	if ok, err := repo.MarkUncollectibleScoped(ctx, tenantID, wo); err != nil || !ok {
		t.Fatalf("MarkUncollectibleScoped: ok=%v err=%v", ok, err)
	}
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id=$1`, wo).Scan(&status); err != nil {
		t.Fatalf("read written-off: %v", err)
	}
	if status != "uncollectible" {
		t.Errorf("status = %q, want uncollectible", status)
	}
	// A second call is a no-op (already uncollectible, not in the collectible set).
	if ok, _ := repo.MarkUncollectibleScoped(ctx, tenantID, wo); ok {
		t.Error("re-writing off an already-uncollectible invoice must be a no-op")
	}

	// --- Cross-tenant safety: another tenant can't touch these invoices. ---
	otherTenant := uuid.New()
	if ok, _ := repo.RequeueForRetry(ctx, otherTenant, pd); ok {
		t.Error("cross-tenant requeue must affect no rows")
	}
	if ok, _ := repo.SetDunningPaused(ctx, otherTenant, pd, true); ok {
		t.Error("cross-tenant pause must affect no rows")
	}
	if ok, _ := repo.MarkUncollectibleScoped(ctx, otherTenant, pd); ok {
		t.Error("cross-tenant write-off must affect no rows")
	}
}

func containsInvoice(invs []*domain.Invoice, id uuid.UUID) bool {
	for _, inv := range invs {
		if inv.ID == id {
			return true
		}
	}
	return false
}
