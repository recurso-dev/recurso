package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCampaignClaim_SkipsPausedInvoice_Postgres proves the QA fix: an operator
// "pause dunning" (Collections Inc 3) silences the CAMPAIGN worker too — a due
// execution on a paused invoice is neither claimed nor listed, its next_step_at
// is left untouched (so un-pausing resumes it on the next tick), and un-pausing
// makes it claimable again.
func TestCampaignClaim_SkipsPausedInvoice_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed campaign-pause test")
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

	tenantID, custID := seedCreditAppTenantCustomer(t, conn)
	invRepo := NewInvoiceRepository(conn).(*InvoiceRepository)
	campRepo := NewDunningCampaignRepository(conn)

	invoiceID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,'USD',10000,10000,0,0,'past_due',$4,NOW(),NOW() - INTERVAL '5 days')`,
		invoiceID, tenantID, custID, "CMP-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	campaignID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO dunning_campaigns (id, tenant_id, name, trigger_event) VALUES ($1,$2,$3,'payment_failed')`,
		campaignID, tenantID, "Camp-"+run); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	execID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO dunning_campaign_executions (id, tenant_id, invoice_id, campaign_id, status, next_step_at)
		 VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '1 hour')`,
		execID, tenantID, invoiceID, campaignID); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	now := time.Now()
	lease := now.Add(15 * time.Minute)

	// Paused → the due execution is invisible to both the claim and the listing.
	if ok, err := invRepo.SetDunningPaused(ctx, tenantID, invoiceID, true); err != nil || !ok {
		t.Fatalf("SetDunningPaused(true): ok=%v err=%v", ok, err)
	}
	claimed, err := campRepo.ClaimDueExecutions(ctx, now, lease, 50)
	if err != nil {
		t.Fatalf("ClaimDueExecutions (paused): %v", err)
	}
	for _, e := range claimed {
		if e.ID == execID {
			t.Fatal("a paused invoice's campaign execution must NOT be claimed")
		}
	}
	due, err := campRepo.GetDueExecutions(ctx, now)
	if err != nil {
		t.Fatalf("GetDueExecutions (paused): %v", err)
	}
	for _, e := range due {
		if e.ID == execID {
			t.Fatal("a paused invoice's campaign execution must NOT be listed as due")
		}
	}
	// Its next_step_at was left in the past (not leased) — un-pausing resumes
	// immediately rather than waiting out a phantom lease.
	var nextStep time.Time
	if err := conn.QueryRowContext(ctx,
		`SELECT next_step_at FROM dunning_campaign_executions WHERE id=$1`, execID).Scan(&nextStep); err != nil {
		t.Fatalf("read next_step_at: %v", err)
	}
	if !nextStep.Before(now) {
		t.Errorf("paused execution's next_step_at moved to %v; skipping must not lease it", nextStep)
	}

	// Un-paused → claimable again.
	if ok, err := invRepo.SetDunningPaused(ctx, tenantID, invoiceID, false); err != nil || !ok {
		t.Fatalf("SetDunningPaused(false): ok=%v err=%v", ok, err)
	}
	claimed2, err := campRepo.ClaimDueExecutions(ctx, now, lease, 50)
	if err != nil {
		t.Fatalf("ClaimDueExecutions (resumed): %v", err)
	}
	found := false
	for _, e := range claimed2 {
		if e.ID == execID {
			found = true
		}
	}
	if !found {
		t.Fatal("after un-pausing, the due execution must be claimable again")
	}
}
