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

// TestUpgradeProration_ScopedToSubscriptionEntity proves the Multi-Entity Books
// fix: an upgrade's proration CHARGE invoice must post to the subscription's own
// legal entity, not the primary. Before the fix the chargeInvoice omitted
// EntityID, so RecordInvoice resolved nil→primary and the proration's AR/Deferred
// legs landed on the primary ledger while the sub's own books never saw them.
//
// The invoice's entity_id column AND the entity_id of its AR ledger account are
// the oracle: both must equal the subscription's (non-primary) entity.
func TestUpgradeProration_ScopedToSubscriptionEntity(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed upgrade entity-scope test")
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
	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]

	// A second, NON-primary legal entity with its own ledger (tb_ledger_id = 2).
	var entityID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
		 VALUES ($1, $2, $2, FALSE, 2, $3) RETURNING id`,
		tenantID, "EU GmbH "+run, "EU-"+run).Scan(&entityID); err != nil {
		t.Fatalf("seed second entity: %v", err)
	}
	// The trigger seeds the invoice sequence only for entities created through the
	// app; a raw INSERT needs it explicitly so the proration invoice can number.
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO entity_invoice_sequences (entity_id, next_number) VALUES ($1, 1)`, entityID); err != nil {
		t.Fatalf("seed entity invoice sequence: %v", err)
	}

	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme US', 'United States', 'individual', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return id
	}
	currentPlanID := seedPlan("Basic", 100000)
	targetPlanID := seedPlan("Pro", 200000)

	// Subscription scoped to the SECOND entity — the whole point of the test.
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, entity_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID, entityID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(db.NewEntityRepository(conn)) // production wires this (main.go); required for per-entity posting
	subRepo := db.NewSubscriptionRepository(conn)
	revrec := NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)
	svc := NewSubscriptionService(subRepo, db.NewInvoiceRepository(conn), db.NewPlanRepository(conn),
		db.NewCustomerRepository(dbx), nil, nil, ledger, nil, nil, db.NewTxManager(conn), revrec, nil)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (upgrade): %v", err)
	}

	// ORACLE 1: the proration invoice's entity_id must be the sub's entity.
	var invID, invEntity uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id, entity_id FROM invoices WHERE subscription_id = $1 AND status <> 'draft' ORDER BY created_at DESC LIMIT 1`,
		subID).Scan(&invID, &invEntity); err != nil {
		t.Fatalf("read proration invoice: %v", err)
	}
	if invEntity != entityID {
		t.Fatalf("proration invoice entity_id = %s, want %s (the subscription's entity)", invEntity, entityID)
	}

	// ORACLE 2: the AR ledger legs for that invoice must sit on that entity's books.
	// Every ledger account touched by the invoice's legs must be entity-scoped.
	var wrongEntityLegs int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions t
		   JOIN ledger_accounts da ON da.id = t.debit_account_id
		   JOIN ledger_accounts ca ON ca.id = t.credit_account_id
		  WHERE t.reference_id = $1
		    AND (da.entity_id IS DISTINCT FROM $2 OR ca.entity_id IS DISTINCT FROM $2)`,
		invID, entityID).Scan(&wrongEntityLegs); err != nil {
		t.Fatalf("check ledger leg entities: %v", err)
	}
	if wrongEntityLegs != 0 {
		t.Fatalf("%d proration ledger leg(s) posted to the wrong entity's accounts (want all on %s)", wrongEntityLegs, entityID)
	}
}
