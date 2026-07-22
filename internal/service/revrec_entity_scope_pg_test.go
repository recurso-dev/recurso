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

// TestRevRecRecognition_ScopedToInvoiceEntity proves the Multi-Entity Books fix
// for revenue recognition. Before it, RecordRecognition hardcoded the primary
// ledger: a non-primary entity's subscription invoice credited Deferred on that
// entity at invoice time, but every recognition event debited Deferred / credited
// Recognized on the PRIMARY entity — so the non-primary Deferred grew forever and
// revenue landed on the wrong entity's P&L.
//
// Oracle: after recognition, the primary entity's Deferred must be untouched, the
// non-primary entity's Deferred must fully drain, and every code-2 recognition leg
// must sit on the non-primary entity's accounts.
func TestRevRecRecognition_ScopedToInvoiceEntity(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed revrec entity-scope test")
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

	// Second, non-primary legal entity with its own ledger (tb_ledger_id = 2).
	var entityID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
		 VALUES ($1, $2, $2, FALSE, 2, $3) RETURNING id`,
		tenantID, "EU GmbH "+run, "EU-"+run).Scan(&entityID); err != nil {
		t.Fatalf("seed second entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(db.NewEntityRepository(conn)) // production wires this (main.go)
	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)

	// A subscription invoice issued by the SECOND entity → credits Deferred on
	// that entity's ledger. No tax, so recognition drains it exactly.
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme EU', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	subID := uuid.New()
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Pro','pro-'||$3,'month',1,TRUE)`,
		planID, tenantID, run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, entity_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'active', NOW() - INTERVAL '13 months', NOW() - INTERVAL '1 month', NOW() - INTERVAL '13 months', NOW(), NOW())`,
		subID, tenantID, customerID, planID, entityID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	invID := uuid.New()
	inv := &domain.Invoice{
		ID: invID, TenantID: tenantID, CustomerID: customerID,
		EntityID: &entityID, SubscriptionID: &subID,
		InvoiceNumber: "EU-1", Total: 120000, Currency: "USD",
	}
	// The schedule FKs to invoices(id), so the row must exist.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, entity_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		 VALUES ($1,$2,$3,$4,$5,'USD',120000,120000,120000,'paid',$6,NOW(),NOW())`,
		invID, tenantID, customerID, subID, entityID, "EU-1"); err != nil {
		t.Fatalf("seed invoice row: %v", err)
	}
	if err := ledger.RecordInvoice(ctx, inv); err != nil {
		t.Fatalf("RecordInvoice: %v", err)
	}

	// Schedule + events over a period that has fully elapsed, so every event is due.
	start := time.Now().UTC().AddDate(0, -13, 0)
	sub := &domain.Subscription{
		ID: subID, TenantID: tenantID,
		CurrentPeriodStart: start, CurrentPeriodEnd: start.AddDate(0, 12, 0),
	}
	if err := revrec.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
		t.Fatalf("CreateScheduleForInvoice: %v", err)
	}

	// The schedule must have inherited the invoice's entity.
	var schedEntity sql.NullString
	if err := conn.QueryRowContext(ctx,
		`SELECT entity_id FROM revenue_schedules WHERE invoice_id = $1`, inv.ID).Scan(&schedEntity); err != nil {
		t.Fatalf("read schedule entity: %v", err)
	}
	if !schedEntity.Valid || schedEntity.String != entityID.String() {
		t.Fatalf("schedule entity_id = %v, want %s", schedEntity, entityID)
	}

	if err := revrec.ProcessDueEvents(ctx); err != nil {
		t.Fatalf("ProcessDueEvents: %v", err)
	}

	// ORACLE 1: every code-2 recognition leg for THIS tenant posted to the second
	// entity's accounts. Scoped by tenant — the shared test DB retains other
	// tenants' rows (each on their own ledger 2).
	var wrongEntityLegs, legCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FILTER (WHERE da.entity_id IS DISTINCT FROM $1 OR ca.entity_id IS DISTINCT FROM $1),
		        COUNT(*)
		   FROM ledger_transactions t
		   JOIN ledger_accounts da ON da.id = t.debit_account_id
		   JOIN ledger_accounts ca ON ca.id = t.credit_account_id
		  WHERE t.code = 2 AND da.tenant_id = $2`,
		entityID, tenantID).Scan(&wrongEntityLegs, &legCount); err != nil {
		t.Fatalf("check recognition leg entities: %v", err)
	}
	if legCount == 0 {
		t.Fatal("no recognition legs posted for this tenant — recognition did not run")
	}
	if wrongEntityLegs != 0 {
		t.Fatalf("%d recognition leg(s) on the wrong entity's accounts", wrongEntityLegs)
	}

	// ORACLE 2: no code-2 leg for this tenant leaked onto a primary (NULL-entity)
	// account — the pre-fix bug posted recognition to the primary ledger.
	var primaryRecognitionLegs int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions t
		   JOIN ledger_accounts da ON da.id = t.debit_account_id
		  WHERE t.code = 2 AND da.tenant_id = $1 AND da.entity_id IS NULL`,
		tenantID).Scan(&primaryRecognitionLegs); err != nil {
		t.Fatalf("check primary recognition legs: %v", err)
	}
	if primaryRecognitionLegs != 0 {
		t.Fatalf("%d recognition leg(s) leaked onto the primary ledger", primaryRecognitionLegs)
	}

	// ORACLE 3: the second entity's Deferred drained to ~0 (invoice credited it
	// 120000, recognition debited the same back out).
	var deferredBalance int64
	if err := conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN t.credit_account_id = a.id THEN t.amount ELSE -t.amount END), 0)
		   FROM ledger_accounts a
		   JOIN ledger_transactions t ON t.debit_account_id = a.id OR t.credit_account_id = a.id
		  WHERE a.entity_id = $1 AND a.code = $2`,
		entityID, domain.AccountCodeDeferredRevenue).Scan(&deferredBalance); err != nil {
		t.Fatalf("compute deferred balance: %v", err)
	}
	if deferredBalance != 0 {
		t.Fatalf("second entity Deferred balance = %d, want 0 (fully recognized)", deferredBalance)
	}
}
