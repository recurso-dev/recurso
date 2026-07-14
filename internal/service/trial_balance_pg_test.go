package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestTrialBalance_Postgres proves the aggregation ties out against the real
// ledger schema: a one-off invoice (DR AR / CR Revenue) followed by its payment
// (DR Cash / CR AR) must leave the trial balance balanced, with Cash and
// Revenue carrying their expected balances and no account flagged abnormal.
func TestTrialBalance_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed trial-balance test")
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
		tenantID, "TB-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		InvoiceNumber: "TB-1", Total: 10000, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	if err := svc.RecordPayment(ctx, inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	tb, err := svc.GetTrialBalance(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTrialBalance: %v", err)
	}

	if !tb.Balanced || tb.TotalDebits != tb.TotalCredits {
		t.Fatalf("trial balance does not tie out: balanced=%v D%d/C%d", tb.Balanced, tb.TotalDebits, tb.TotalCredits)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d (%s) abnormal: balance=%d", l.Code, l.Name, l.Balance)
		}
	}

	cash := findLine(t, tb, domain.AccountCodeCash)
	if cash.Balance != 10000 {
		t.Errorf("Cash balance = %d, want 10000", cash.Balance)
	}
	revenue := findLine(t, tb, domain.AccountCodeRevenue)
	if revenue.Balance != 10000 {
		t.Errorf("Revenue balance = %d, want 10000", revenue.Balance)
	}
	ar := findLine(t, tb, domain.AccountCodeAR)
	if ar.Balance != 0 {
		t.Errorf("AR balance = %d, want 0 (invoice then paid)", ar.Balance)
	}
}
