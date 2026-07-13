package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type dcNoopEmail struct{}

func (dcNoopEmail) Send(_ context.Context, _ port.EmailMessage) error { return nil }

// TestDunningCampaignService_ProcessDueSteps_Postgres drives the dunning-campaign
// worker loop against a real DB: a due active execution advances one step and is
// rescheduled for the next. Guards the multi-channel recovery loop (including the
// nested tenant-context injection for the customer read).
func TestDunningCampaignService_ProcessDueSteps_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed dunning-campaign test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	sqlxConn := sqlx.NewDb(conn, "postgres")
	ctx := context.Background()

	tenantID := uuid.New()
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "DC-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "Campaign Cust", "dc-"+customerID.String()[:8]+"@example.com", uuid.New())
	invID := uuid.New()
	mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',10000,10000,0,0,'open',$4, NOW(), NOW() - INTERVAL '3 days')`,
		invID, tenantID, customerID, "INV-DC-"+invID.String()[:8])
	campaignID := uuid.New()
	mustExec(t, conn, `INSERT INTO dunning_campaigns (id, tenant_id, name, is_active, trigger_event, created_at, updated_at)
		VALUES ($1,$2,'Recovery',TRUE,'payment_failed',NOW(),NOW())`, campaignID, tenantID)
	// Two email steps, so the first execution advances to the second (not exhausted).
	mustExec(t, conn, `INSERT INTO dunning_campaign_steps (id, campaign_id, step_order, channel, delay_hours, subject, body, created_at)
		VALUES ($1,$2,0,'email',0,'Reminder','Please pay',NOW())`, uuid.New(), campaignID)
	mustExec(t, conn, `INSERT INTO dunning_campaign_steps (id, campaign_id, step_order, channel, delay_hours, subject, body, created_at)
		VALUES ($1,$2,1,'email',48,'Final notice','Overdue',NOW())`, uuid.New(), campaignID)
	execID := uuid.New()
	mustExec(t, conn, `INSERT INTO dunning_campaign_executions (id, tenant_id, invoice_id, campaign_id, current_step_index, status, started_at, next_step_at)
		VALUES ($1,$2,$3,$4,0,'active',NOW(), NOW() - INTERVAL '1 minute')`, execID, tenantID, invID, campaignID)

	svc := NewDunningCampaignService(
		db.NewDunningCampaignRepository(conn),
		db.NewInvoiceRepository(conn),
		db.NewCustomerRepository(sqlxConn),
		NewNotificationService(dcNoopEmail{}, "http://example.test"),
		nil, // smsSender
	)

	if err := svc.ProcessDueSteps(ctx); err != nil {
		t.Fatalf("ProcessDueSteps: %v", err)
	}

	var stepIndex int
	var status string
	var nextStep sql.NullTime
	if err := conn.QueryRowContext(ctx,
		`SELECT current_step_index, status, next_step_at FROM dunning_campaign_executions WHERE id = $1`, execID).
		Scan(&stepIndex, &status, &nextStep); err != nil {
		t.Fatalf("read execution: %v", err)
	}
	if stepIndex != 1 {
		t.Errorf("current_step_index = %d, want 1 (should have advanced one step)", stepIndex)
	}
	if status != "active" {
		t.Errorf("status = %q, want active (a second step remains)", status)
	}
	if !nextStep.Valid {
		t.Error("next_step_at should be set for the remaining step")
	}
}
