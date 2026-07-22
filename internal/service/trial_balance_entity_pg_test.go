package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestTrialBalance_PerEntityAndConsolidated_Postgres proves Multi-Entity Books
// Inc 4: the trial balance can be scoped to one entity's ledger, and the
// consolidated view sums the same account code across every entity ledger.
func TestTrialBalance_PerEntityAndConsolidated_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed trial-balance-entity test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "TB-"+run, "tb-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	entityRepo := db.NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)

	// A wallet top-up on the primary and one on the second entity: each posts
	// DR Cash / CR Customer Credit on its own ledger.
	if _, err := ledger.RecordWalletTopUp(ctx, tenantID, nil, uuid.New(), 10000, "primary top-up"); err != nil {
		t.Fatalf("primary top-up: %v", err)
	}
	if _, err := ledger.RecordWalletTopUp(ctx, tenantID, &second.ID, uuid.New(), 25000, "UK top-up"); err != nil {
		t.Fatalf("UK top-up: %v", err)
	}

	cashDebits := func(tb *domain.TrialBalance) int64 {
		var total int64
		for _, l := range tb.Lines {
			if l.Code == domain.AccountCodeCash {
				total += l.Debits
			}
		}
		return total
	}

	// All-entities (nil): both entities' Cash accounts appear, so debits sum.
	allTB, err := ledger.GetTrialBalance(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetTrialBalance(all): %v", err)
	}
	if got := cashDebits(allTB); got != 35000 {
		t.Errorf("all-entities Cash debits = %d, want 35000 (10000 + 25000)", got)
	}

	// Scoped to the second entity: only its 25000.
	ukTB, err := ledger.GetTrialBalance(ctx, tenantID, &second.ID)
	if err != nil {
		t.Fatalf("GetTrialBalance(UK): %v", err)
	}
	if got := cashDebits(ukTB); got != 25000 {
		t.Errorf("UK Cash debits = %d, want 25000", got)
	}
	for _, l := range ukTB.Lines {
		if l.EntityID == nil || *l.EntityID != second.ID {
			t.Errorf("UK-scoped line %s not tagged to the UK entity: %+v", l.Name, l.EntityID)
		}
	}

	// Consolidated: exactly one Cash line, summed across both ledgers.
	cons, err := ledger.GetConsolidatedTrialBalance(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetConsolidatedTrialBalance: %v", err)
	}
	cashLines := 0
	for _, l := range cons.Lines {
		if l.Code == domain.AccountCodeCash {
			cashLines++
			if l.Debits != 35000 {
				t.Errorf("consolidated Cash debits = %d, want 35000", l.Debits)
			}
		}
	}
	if cashLines != 1 {
		t.Errorf("consolidated view has %d Cash lines, want exactly 1 (code-rolled-up)", cashLines)
	}
	if !cons.Balanced {
		t.Errorf("consolidated trial balance is not balanced")
	}
}
