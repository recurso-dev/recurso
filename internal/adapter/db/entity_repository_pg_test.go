package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestEntityRepository_Postgres exercises the Multi-Entity migration (000128):
// the primary-entity trigger fires on tenant insert, the repo allocates ledger
// ids and seeds invoice sequences, and the one-primary constraint holds.
func TestEntityRepository_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	ctx := context.Background()
	repo := NewEntityRepository(conn)

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	// Inserting a tenant must auto-create its primary entity via the trigger.
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "ME-"+run, "me-"+run+"@t.com")

	primary, err := repo.GetPrimary(ctx, tenantID)
	if err != nil {
		t.Fatalf("get primary: %v", err)
	}
	if primary == nil {
		t.Fatal("trigger did not create a primary entity on tenant insert")
	}
	if !primary.IsPrimary || primary.TBLedgerID != 1 {
		t.Errorf("primary entity: is_primary=%v tb_ledger_id=%d, want true/1", primary.IsPrimary, primary.TBLedgerID)
	}
	// Its invoice sequence row exists and starts at 1.
	var next int64
	if err := conn.QueryRowContext(ctx,
		`SELECT next_number FROM entity_invoice_sequences WHERE entity_id = $1`, primary.ID).Scan(&next); err != nil {
		t.Fatalf("primary sequence not seeded: %v", err)
	}
	if next != 1 {
		t.Errorf("primary sequence next_number = %d, want 1", next)
	}

	// Create a second entity — the repo allocates the next ledger id + a sequence.
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", LegalName: "ACME UK Ltd", InvoicePrefix: "ACME-UK", CountryCode: "GB"}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second entity: %v", err)
	}
	if second.TBLedgerID != 2 {
		t.Errorf("second entity tb_ledger_id = %d, want 2 (next after primary)", second.TBLedgerID)
	}
	if err := conn.QueryRowContext(ctx,
		`SELECT next_number FROM entity_invoice_sequences WHERE entity_id = $1`, second.ID).Scan(&next); err != nil {
		t.Fatalf("second entity sequence not seeded: %v", err)
	}

	list, err := repo.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(list))
	}
	if !list[0].IsPrimary {
		t.Error("List should return the primary entity first")
	}

	// The one-primary partial unique index must reject a second primary.
	_, err = conn.ExecContext(ctx,
		`INSERT INTO entities (tenant_id, name, is_primary, tb_ledger_id) VALUES ($1,'dup',TRUE,9)`, tenantID)
	if err == nil {
		t.Error("a tenant must not be allowed two primary entities")
	}
}
