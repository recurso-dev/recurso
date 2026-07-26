package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestGSTR1Export_PerEntity proves the per-entity GSTR filter: a concrete entity
// id returns only that entity's invoices/credit notes, while nil returns the
// whole tenant (the single-entity / consolidated behavior, unchanged).
func TestGSTR1Export_PerEntity(t *testing.T) {
	conn := openGSTR1TestDB(t)
	repo := NewInvoiceRepository(conn).(*InvoiceRepository)
	ctx := context.Background()

	tenantID, _ := seedCreditAppTenantCustomer(t, conn)
	cust := seedGSTCustomer(t, conn, tenantID, "27AAAAA0000A1Z5", "27")

	// Entity A = the tenant's existing primary (auto-created); B = a new branch.
	var entA, entB uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM entities WHERE tenant_id=$1 AND is_primary=TRUE`, tenantID).Scan(&entA); err != nil {
		t.Fatalf("load primary entity: %v", err)
	}
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, is_primary, tb_ledger_id, invoice_prefix) VALUES ($1,'Branch',FALSE,2,$2) RETURNING id`,
		tenantID, "BR"+uuid.New().String()[:4]).Scan(&entB); err != nil {
		t.Fatalf("seed entity B: %v", err)
	}

	july := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	// Two invoices for A, one for B.
	seedGSTInvoice(t, conn, tenantID, cust, "E-A-1", "open", "9983", 100000, 0, 9000, 9000, july)
	seedGSTInvoice(t, conn, tenantID, cust, "E-A-2", "open", "9983", 50000, 0, 4500, 4500, july)
	invB1 := seedGSTInvoice(t, conn, tenantID, cust, "E-B-1", "open", "9983", 20000, 0, 1800, 1800, july)
	// Stamp entity_id (the seed helper leaves it NULL).
	stamp := func(number string, ent uuid.UUID) {
		if _, err := conn.ExecContext(ctx, `UPDATE invoices SET entity_id=$1 WHERE invoice_number=$2 AND tenant_id=$3`, ent, number, tenantID); err != nil {
			t.Fatalf("stamp %s: %v", number, err)
		}
	}
	stamp("E-A-1", entA)
	stamp("E-A-2", entA)
	stamp("E-B-1", entB)

	// A refund credit note for B, stamped to entity B.
	seedRefundCreditNote(t, conn, tenantID, cust, invB1, "CN-B-1", 5000, july)
	if _, err := conn.ExecContext(ctx, `UPDATE credit_notes SET entity_id=$1 WHERE reference='CN-B-1'`, entB); err != nil {
		t.Fatalf("stamp CN-B: %v", err)
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	// Entity A: only its two invoices, no credit notes.
	aInv, err := repo.GetGSTR1Invoices(ctx, tenantID, &entA, start, end)
	if err != nil {
		t.Fatalf("GetGSTR1Invoices(A): %v", err)
	}
	if len(aInv) != 2 {
		t.Errorf("entity A invoices = %d, want 2", len(aInv))
	}
	aCN, _ := repo.GetGSTR1CreditNotes(ctx, tenantID, &entA, start, end)
	if len(aCN) != 0 {
		t.Errorf("entity A credit notes = %d, want 0", len(aCN))
	}

	// Entity B: its one invoice and one credit note.
	bInv, err := repo.GetGSTR1Invoices(ctx, tenantID, &entB, start, end)
	if err != nil {
		t.Fatalf("GetGSTR1Invoices(B): %v", err)
	}
	if len(bInv) != 1 || bInv[0].InvoiceNumber != "E-B-1" {
		t.Errorf("entity B invoices = %+v, want just E-B-1", bInv)
	}
	bCN, _ := repo.GetGSTR1CreditNotes(ctx, tenantID, &entB, start, end)
	if len(bCN) != 1 {
		t.Errorf("entity B credit notes = %d, want 1", len(bCN))
	}

	// nil = whole tenant (all three invoices).
	allInv, err := repo.GetGSTR1Invoices(ctx, tenantID, nil, start, end)
	if err != nil {
		t.Fatalf("GetGSTR1Invoices(nil): %v", err)
	}
	if len(allInv) != 3 {
		t.Errorf("consolidated invoices = %d, want 3", len(allInv))
	}
}
