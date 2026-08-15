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

// TestReconciliationRun_GetByID_Postgres proves a recorded run is addressable
// with its persisted discrepancy rows (the per-run detail that makes a
// historical run explainable), and that the read is tenant-scoped.
func TestReconciliationRun_GetByID_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping reconciliation-run test")
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

	seedTenant := func() uuid.UUID {
		id := uuid.New()
		if _, err := database.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
			id, "RR-"+id.String()[:8], id.String()[:8]+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
		return id
	}
	tenantID := seedTenant()
	repo := NewReconciliationRunRepository(database)

	invID := uuid.New()
	run := &domain.ReconciliationRun{
		TenantID: tenantID, RunAt: time.Now().UTC(),
		InvoicesChecked: 12, PaidInvoicesChecked: 8, TotalDiscrepancies: 2,
	}
	discrepancies := []domain.ReconciliationRunDiscrepancy{
		{Type: "invoice_amount_mismatch", InvoiceID: &invID, ExpectedAmount: 10000, FoundAmount: 9000},
		{Type: "ledger_unbalanced", ExpectedAmount: 500, FoundAmount: 0},
	}
	if err := repo.Create(ctx, run, discrepancies); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Owned read returns the summary + both discrepancy rows, in insertion order.
	got, err := repo.GetByID(ctx, tenantID, run.ID)
	if err != nil {
		t.Fatalf("GetByID(owned): %v", err)
	}
	if got == nil {
		t.Fatal("GetByID(owned) returned nil, want the run")
	}
	if got.InvoicesChecked != 12 || got.TotalDiscrepancies != 2 {
		t.Errorf("run summary not round-tripped: %+v", got.ReconciliationRun)
	}
	if len(got.Discrepancies) != 2 {
		t.Fatalf("expected 2 stored discrepancies, got %d", len(got.Discrepancies))
	}
	if got.Discrepancies[0].Type != "invoice_amount_mismatch" || got.Discrepancies[0].InvoiceID == nil ||
		*got.Discrepancies[0].InvoiceID != invID || got.Discrepancies[0].ExpectedAmount != 10000 {
		t.Errorf("first discrepancy not round-tripped: %+v", got.Discrepancies[0])
	}
	if got.DiscrepanciesTruncated {
		t.Error("all discrepancies were stored; DiscrepanciesTruncated should be false")
	}

	// Cross-tenant read returns nothing.
	foreign, err := repo.GetByID(ctx, seedTenant(), run.ID)
	if err != nil {
		t.Fatalf("GetByID(cross-tenant): %v", err)
	}
	if foreign != nil {
		t.Error("cross-tenant read returned a run; must be nil (404)")
	}

	// Unknown id: nil, no error.
	if missing, err := repo.GetByID(ctx, tenantID, uuid.New()); err != nil || missing != nil {
		t.Errorf("unknown id: got (%v, %v), want (nil, nil)", missing, err)
	}

	// A clean run persists no discrepancy rows but is still addressable.
	clean := &domain.ReconciliationRun{TenantID: tenantID, RunAt: time.Now().UTC(), TotalDiscrepancies: 0}
	if err := repo.Create(ctx, clean, nil); err != nil {
		t.Fatalf("Create(clean): %v", err)
	}
	cleanGot, err := repo.GetByID(ctx, tenantID, clean.ID)
	if err != nil || cleanGot == nil {
		t.Fatalf("GetByID(clean): %v (got=%v)", err, cleanGot)
	}
	if len(cleanGot.Discrepancies) != 0 || cleanGot.DiscrepanciesTruncated {
		t.Errorf("clean run should have an empty, non-truncated discrepancy list, got %+v", cleanGot)
	}
}
