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

// TestLedger_PerEntityIsolation_Postgres proves Multi-Entity Books Inc 2a: an
// invoice issued by a non-primary entity posts to that entity's own ledger
// (tb_ledger_id) and its own chart of accounts — including a separate AR
// sub-ledger — so the SAME customer's receivables are isolated per entity, and
// the primary entity's books are untouched.
func TestLedger_PerEntityIsolation_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed entity-isolation test")
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
		tenantID, "ME2-"+run, "me2-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	entityRepo := db.NewEntityRepository(conn)
	primary, err := entityRepo.GetPrimary(ctx, tenantID)
	if err != nil || primary == nil {
		t.Fatalf("primary entity missing: %v", err)
	}
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create second entity: %v", err)
	}
	if second.TBLedgerID != 2 {
		t.Fatalf("second entity ledger id = %d, want 2", second.TBLedgerID)
	}

	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1,$2,$3,'Acme',' ','individual',$4,NOW(),NOW())`,
		custID, tenantID, "c-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)

	// A one-off invoice ISSUED BY THE SECOND ENTITY (Total 100000, no tax).
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, entity_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,'USD',100000,100000,100000,0,'paid',$5,NOW(),NOW())`,
		invID, tenantID, second.ID, custID, "UK-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	inv := &domain.Invoice{
		ID: invID, TenantID: tenantID, EntityID: &second.ID, CustomerID: custID,
		InvoiceNumber: "UK-" + run, Total: 100000, Currency: "USD",
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}
	if err := ledger.RecordPayment(ctx, inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	// Every posting for this invoice is on the second entity's ledger (id 2).
	var wrongLedger int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1 AND ledger_id <> 2`, invID).Scan(&wrongLedger); err != nil {
		t.Fatalf("count postings: %v", err)
	}
	if wrongLedger != 0 {
		t.Errorf("%d postings landed off the entity's ledger (id 2)", wrongLedger)
	}

	// The second entity's Revenue account exists on ledger 2, tagged to the entity.
	var revLedger int
	var revEntity uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT ledger_id, entity_id FROM ledger_accounts WHERE tenant_id=$1 AND entity_id=$2 AND code=$3`,
		tenantID, second.ID, domain.AccountCodeRevenue).Scan(&revLedger, &revEntity); err != nil {
		t.Fatalf("second entity revenue account missing: %v", err)
	}
	if revLedger != 2 || revEntity != second.ID {
		t.Errorf("revenue account: ledger=%d entity=%s, want 2 / %s", revLedger, revEntity, second.ID)
	}

	// The second entity's AR account for this customer is a DIFFERENT account id
	// than the customer id (which is the primary entity's AR id), on ledger 2.
	arID := ledger.arAccountID(ledgerEntity{ID: second.ID, LedgerID: 2, Primary: false}, custID)
	if arID == custID {
		t.Fatal("non-primary AR id must differ from the customer id (primary AR)")
	}
	var arLedger int
	if err := conn.QueryRowContext(ctx,
		`SELECT ledger_id FROM ledger_accounts WHERE id=$1 AND tenant_id=$2 AND code=$3`,
		arID, tenantID, domain.AccountCodeAR).Scan(&arLedger); err != nil {
		t.Fatalf("second entity AR account missing: %v", err)
	}
	if arLedger != 2 {
		t.Errorf("second entity AR on ledger %d, want 2", arLedger)
	}

	// The primary entity's AR account (id = customer id) has NO postings — the
	// receivable lives entirely in the second entity's books.
	var primaryARDebits int
	_ = conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE (debit_account_id=$1 OR credit_account_id=$1)`, custID).Scan(&primaryARDebits)
	if primaryARDebits != 0 {
		t.Errorf("the primary AR account (customer id) received %d postings; it must stay isolated from the second entity", primaryARDebits)
	}
}
