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

// TestScheduleAccountingVersion_Postgres proves the accounting-model version
// (ADR-008) persists and round-trips on revenue_schedules: an explicit V2
// (accrual) schedule reads back V2, and a schedule created without a version
// (legacy call path, zero value) defaults to V1 (cash) via the repo + the
// column default.
func TestScheduleAccountingVersion_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed accounting-version test")
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

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "AV-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewRevRecRepository(conn)

	seedInvoice := func() uuid.UUID {
		invID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, tax_type, status, created_at, updated_at)
			 VALUES ($1, $2, 'USD', 10000, 10000, $3, 'none', 'open', NOW(), NOW())`,
			invID, tenantID, "INV-AV-"+invID.String()[:8]); err != nil {
			t.Fatalf("seed invoice: %v", err)
		}
		return invID
	}

	// Explicit V2 (accrual) schedule round-trips as V2.
	invV2 := seedInvoice()
	if err := repo.CreateSchedule(ctx, &domain.RevenueSchedule{
		ID: uuid.New(), TenantID: tenantID, InvoiceID: invV2,
		TotalAmount: 10000, Currency: "USD", StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
		Status: "active", AccountingVersion: domain.AccountingModelV2,
	}); err != nil {
		t.Fatalf("create V2 schedule: %v", err)
	}
	got, err := repo.GetActiveScheduleByInvoice(ctx, tenantID, invV2)
	if err != nil || got == nil {
		t.Fatalf("read V2 schedule: got=%v err=%v", got, err)
	}
	if got.AccountingVersion != domain.AccountingModelV2 {
		t.Errorf("V2 schedule read back version %d, want %d", got.AccountingVersion, domain.AccountingModelV2)
	}

	// Unset version (legacy caller, zero value) defaults to V1 via the repo.
	invV1 := seedInvoice()
	if err := repo.CreateSchedule(ctx, &domain.RevenueSchedule{
		ID: uuid.New(), TenantID: tenantID, InvoiceID: invV1,
		TotalAmount: 10000, Currency: "USD", StartDate: time.Now(), EndDate: time.Now().AddDate(0, 1, 0),
		Status: "active", // AccountingVersion left 0
	}); err != nil {
		t.Fatalf("create default schedule: %v", err)
	}
	got, err = repo.GetActiveScheduleByInvoice(ctx, tenantID, invV1)
	if err != nil || got == nil {
		t.Fatalf("read default schedule: got=%v err=%v", got, err)
	}
	if got.AccountingVersion != domain.AccountingModelV1 {
		t.Errorf("unset version defaulted to %d, want %d (V1)", got.AccountingVersion, domain.AccountingModelV1)
	}
}
