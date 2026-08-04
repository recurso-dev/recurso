package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestRecognitionOverrun_Postgres proves the accrual invariant
// "Revenue Recognized ≤ the recognizable amount" is really enforced against
// Postgres: a schedule whose recognized events sum to MORE than its total is
// surfaced by the reconciler as recognized_exceeds_invoice, and a healthy
// schedule (recognized ≤ total) is not. This is the "verify by neutering"
// discipline — we make the books wrong on purpose and prove the oracle fires.
func TestRecognitionOverrun_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed recognition-overrun test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "RO-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := db.NewRevRecRepository(conn)
	recon := NewReconciliationService(db.NewLedgerRepository(conn), nil)

	// seedSchedule inserts an invoice + schedule with `total` recognizable and a
	// single recognized event of `recognized`.
	seedSchedule := func(total, recognized int64) uuid.UUID {
		invID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, tax_type, status, created_at, updated_at)
			 VALUES ($1, $2, 'USD', $3, $3, $4, 'none', 'open', NOW(), NOW())`,
			invID, tenantID, total, "INV-RO-"+invID.String()[:8]); err != nil {
			t.Fatalf("seed invoice: %v", err)
		}
		sched := &domain.RevenueSchedule{
			ID: uuid.New(), TenantID: tenantID, InvoiceID: invID,
			TotalAmount: total, Currency: "USD", StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
			Status: "active",
		}
		if err := repo.CreateSchedule(ctx, sched); err != nil {
			t.Fatalf("create schedule: %v", err)
		}
		if err := repo.CreateEvents(ctx, []*domain.RecognitionEvent{
			{ID: uuid.New(), RevenueScheduleID: sched.ID, TenantID: tenantID, Amount: recognized, RecognitionDate: time.Now(), Status: "recognized"},
		}); err != nil {
			t.Fatalf("create events: %v", err)
		}
		return invID
	}

	// Healthy schedule: recognized (7,000) ≤ total (10,000) — must NOT flag.
	seedSchedule(10000, 7000)
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("recon (healthy): %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyRecognizedExceedsInvoice {
			t.Fatalf("healthy schedule wrongly flagged as overrun: %+v", d)
		}
	}

	// Over-recognized schedule: recognized (12,000) > total (10,000) — must flag.
	badInv := seedSchedule(10000, 12000)
	report, err = recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("recon (overrun): %v", err)
	}
	var found *ReconciliationDiscrepancy
	for i := range report.Discrepancies {
		if report.Discrepancies[i].Type == DiscrepancyRecognizedExceedsInvoice {
			found = &report.Discrepancies[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("over-recognized schedule NOT flagged; discrepancies=%+v", report.Discrepancies)
	}
	if found.InvoiceID == nil || *found.InvoiceID != badInv {
		t.Errorf("flagged InvoiceID = %v, want %v", found.InvoiceID, badInv)
	}
	if found.ExpectedAmount != 10000 || found.FoundAmount != 12000 {
		t.Errorf("amounts = recognizable %d / recognized %d, want 10000 / 12000", found.ExpectedAmount, found.FoundAmount)
	}
}
