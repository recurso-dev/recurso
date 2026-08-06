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

// TestLedger_PrimaryPostingRoutedToPrimaryLedger_Postgres pins a Multi-Entity
// Books ledger-routing bug: getOrCreateTenantAccount (the PRIMARY posting path)
// looked up its account with GetAccountByTenantAndCode, which had no entity/ledger
// filter and no ORDER BY. In a tenant that already has a NON-primary entity's
// account for a code, that lookup could return the non-primary account, so a
// primary invoice's Revenue/Deferred landed on the WRONG entity's ledger. The
// books still balance globally (double-entry), so no other reconciler check
// notices — only per-entity attribution is wrong.
//
// The test forces the failure order deterministically: it funds the SECOND
// entity's Deferred FIRST (so the only code-2100 account in the tenant is the
// non-primary one), then posts a PRIMARY subscription invoice. With the bug the
// primary Deferred lands on the second entity's ledger 2; the fix (entity_id IS
// NULL filter) makes the primary posting create/use its own ledger-1 account.
func TestLedger_PrimaryPostingRoutedToPrimaryLedger_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed account-routing test")
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

	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	var entity2 uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
		 VALUES ($1, $2, $2, FALSE, 2, $3) RETURNING id`,
		tenantID, "EU GmbH "+run, "EU-"+run).Scan(&entity2); err != nil {
		t.Fatalf("seed second entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(db.NewEntityRepository(conn))

	// A subscription invoice that funds Deferred on the given entity (nil ⇒ primary).
	fund := func(entityID *uuid.UUID, total int64, tag string) uuid.UUID {
		custID := uuid.New()
		mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
			VALUES ($1,$2,$3,'Cust '||$4,$5,NOW(),NOW())`,
			custID, tenantID, "route-"+run+"-"+tag+"@t.com", tag, uuid.New())
		subID := uuid.New()
		planID := uuid.New()
		mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active)
			VALUES ($1,$2,'Pro','pro-'||$3,'month',1,TRUE)`, planID, tenantID, run+tag)
		mustExec(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, entity_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'active', NOW() - INTERVAL '1 month', NOW(), NOW() - INTERVAL '1 month', NOW(), NOW())`,
			subID, tenantID, custID, planID, entityID)
		invID := uuid.New()
		invNo := "ROUTE-" + run + "-" + tag
		mustExec(t, conn, `INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, entity_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
			VALUES ($1,$2,$3,$4,$5,'USD',$6,$6,0,'open',$7,NOW(),NOW())`,
			invID, tenantID, custID, subID, entityID, total, invNo)
		inv := &domain.Invoice{
			ID: invID, TenantID: tenantID, CustomerID: custID,
			EntityID: entityID, SubscriptionID: &subID,
			InvoiceNumber: invNo, Total: total, Currency: "USD",
		}
		if err := ledger.RecordInvoice(ctx, inv); err != nil {
			t.Fatalf("RecordInvoice (%s): %v", tag, err)
		}
		return invID
	}

	// 1) Second entity funds Deferred FIRST → the only code-2100 account so far is
	//    the NON-primary one (ledger 2).
	fund(&entity2, 90000, "eu")
	// 2) Primary entity funds Deferred → must NOT reuse the second entity's account.
	fund(nil, 40000, "primary")

	// The primary entity's Deferred must sit on the primary ledger (1), entity_id
	// NULL, and hold exactly the primary invoice's amount; the second entity's
	// Deferred (ledger 2) must hold only its own.
	var primaryLedgerDeferred, entity2LedgerDeferred int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(credits_posted - debits_posted),0) FROM ledger_accounts
		 WHERE tenant_id=$1 AND code=$2 AND entity_id IS NULL AND ledger_id=1`,
		tenantID, domain.AccountCodeDeferredRevenue).Scan(&primaryLedgerDeferred); err != nil {
		t.Fatalf("query primary deferred: %v", err)
	}
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(credits_posted - debits_posted),0) FROM ledger_accounts
		 WHERE tenant_id=$1 AND code=$2 AND entity_id=$3`,
		tenantID, domain.AccountCodeDeferredRevenue, entity2).Scan(&entity2LedgerDeferred); err != nil {
		t.Fatalf("query entity2 deferred: %v", err)
	}

	if primaryLedgerDeferred != 40000 {
		t.Errorf("primary-ledger Deferred = %d, want 40000 (the primary invoice) — a primary posting was misrouted to another entity's ledger",
			primaryLedgerDeferred)
	}
	if entity2LedgerDeferred != 90000 {
		t.Errorf("second-entity Deferred = %d, want 90000 (only its own invoice) — a primary posting leaked onto its ledger",
			entity2LedgerDeferred)
	}
}
