package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestEUConfigAndEInvoiceRepos_Postgres exercises the Track C migration (000119):
// the tenant EU config upserts + reads back, and an eu_einvoices record
// round-trips and is idempotent on invoice_id (re-upsert overwrites, no dup).
func TestEUConfigAndEInvoiceRepos_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	ctx := context.Background()

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EU-"+run, "eu-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New())
	invID := uuid.New()
	must(t, conn, `INSERT INTO invoices
		(id, tenant_id, customer_id, currency, subtotal, tax_amount, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'EUR',100000,21000,121000,0,0,'open',$4,NOW(),NOW())`,
		invID, tenantID, custID, "INV-EU-"+run)

	// --- tenant EU config ---
	cfgRepo := NewTenantEUConfigRepository(conn)
	if got, err := cfgRepo.GetByTenantID(ctx, tenantID); err != nil || got != nil {
		t.Fatalf("expected no config yet, got (%v,%v)", got, err)
	}
	cfg := &domain.TenantEUConfig{
		TenantID: tenantID, Enabled: true, LegalName: "Acme GmbH", VATNumber: "DE123456789",
		CountryCode: "DE", Street: "Hauptstr. 1", City: "Berlin", PostalZone: "10115",
	}
	if err := cfgRepo.Upsert(ctx, nil, cfg); err != nil {
		t.Fatalf("upsert config: %v", err)
	}
	got, err := cfgRepo.GetByTenantID(ctx, tenantID)
	if err != nil || got == nil {
		t.Fatalf("get config: %v", err)
	}
	if !got.Enabled || got.VATNumber != "DE123456789" || got.CountryCode != "DE" {
		t.Fatalf("config round-trip wrong: %+v", got)
	}
	// Re-upsert flips the opt-in flag in place.
	cfg.Enabled = false
	_ = cfgRepo.Upsert(ctx, nil, cfg)
	got, _ = cfgRepo.GetByTenantID(ctx, tenantID)
	if got.Enabled {
		t.Fatal("re-upsert should have disabled the config")
	}

	// --- eu_einvoices record ---
	eiRepo := NewEUInvoiceRepository(conn)
	rec := &domain.EUInvoice{
		TenantID: tenantID, InvoiceID: invID, Syntax: domain.EUInvoiceSyntaxUBL,
		Status: domain.EUInvoiceStatusGenerated, Document: "<Invoice/>",
	}
	if err := eiRepo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert e-invoice: %v", err)
	}
	// Idempotent on invoice_id: a second upsert (now 'sent' with a message id)
	// overwrites the same row rather than inserting a duplicate.
	rec2 := &domain.EUInvoice{
		TenantID: tenantID, InvoiceID: invID, Syntax: domain.EUInvoiceSyntaxUBL,
		Status: domain.EUInvoiceStatusSent, Document: "<Invoice/>", MessageID: "mock-abc123",
	}
	if err := eiRepo.Upsert(ctx, rec2); err != nil {
		t.Fatalf("re-upsert e-invoice: %v", err)
	}
	back, err := eiRepo.GetByInvoiceID(ctx, invID)
	if err != nil || back == nil {
		t.Fatalf("get e-invoice: %v", err)
	}
	if back.Status != domain.EUInvoiceStatusSent || back.MessageID != "mock-abc123" {
		t.Fatalf("e-invoice round-trip wrong: %+v", back)
	}

	var count int
	must0(t, conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM eu_einvoices WHERE invoice_id = $1`, invID).Scan(&count))
	if count != 1 {
		t.Fatalf("want exactly 1 eu_einvoices row (idempotent), got %d", count)
	}
}
