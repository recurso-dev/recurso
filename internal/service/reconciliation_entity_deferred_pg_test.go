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

// TestReconciliation_PerEntityDeferred_Postgres proves the R-015 fix against real
// Postgres and the real Multi-Entity Books schema: the deferred-vs-scheduled
// invariant must be evaluated PER ENTITY, and the primary/non-primary entity keys
// must align across the two data sources it joins —
//   - Deferred balance: trial-balance lines, whose EntityID the trial-balance
//     query resolves from the account's ledger_id (the primary account resolves to
//     the primary entity's UUID);
//   - pending recognition: recognition_events joined to revenue_schedules.entity_id,
//     which is NULL for primary-entity schedules (the NULL⇒primary convention).
//
// Phase 1 (no false positive): two legitimately-funded entities — the PRIMARY
// (Deferred 1000, scheduled 40) and a SECOND entity (Deferred 50, scheduled 50) —
// must reconcile clean. This is the landmine guard: a naive per-entity check that
// didn't normalize NULL⇔primary-UUID would key the primary entity differently on
// each side and false-positive here.
//
// Phase 2 (masking caught): bump the SECOND entity's schedule to 100 while its
// Deferred stays 50 — a real per-entity shortfall. The tenant-wide aggregate
// (pending 140 ≤ deferred 1050) would MISS it; the per-entity check must flag
// exactly that entity's shortfall (scheduled 100 / deferred 50).
func TestReconciliation_PerEntityDeferred_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed per-entity deferred test")
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

	// A second, non-primary legal entity on its own ledger (tb_ledger_id = 2).
	var entity2 uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix)
		 VALUES ($1, $2, $2, FALSE, 2, $3) RETURNING id`,
		tenantID, "EU GmbH "+run, "EU-"+run).Scan(&entity2); err != nil {
		t.Fatalf("seed second entity: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(db.NewEntityRepository(conn)) // production wires this (main.go)
	recon := NewReconciliationService(db.NewLedgerRepository(conn), nil)

	// seedFundedEntity posts a subscription invoice (DR AR / CR Deferred on the
	// invoice's entity) and returns the invoice id so a schedule can hang off it.
	// entityID nil ⇒ primary entity; the invoice funds Deferred = total.
	seedFundedEntity := func(entityID *uuid.UUID, total int64, tag string) uuid.UUID {
		custID := uuid.New()
		mustExec(t, conn, `INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
			VALUES ($1,$2,$3,'Cust '||$4,$5,NOW(),NOW())`,
			custID, tenantID, "def-"+run+"-"+tag+"@t.com", tag, uuid.New())
		planID := uuid.New()
		mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active)
			VALUES ($1,$2,'Pro','pro-'||$3,'month',1,TRUE)`, planID, tenantID, run+tag)
		subID := uuid.New()
		mustExec(t, conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, entity_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'active', NOW() - INTERVAL '1 month', NOW(), NOW() - INTERVAL '1 month', NOW(), NOW())`,
			subID, tenantID, custID, planID, entityID)
		invID := uuid.New()
		invNo := "DEF-" + run + "-" + tag
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

	// schedulePending attaches a revenue schedule (tagged to the given entity, NULL
	// for primary) with a single PENDING recognition event of `amount`.
	schedulePending := func(invID uuid.UUID, entityID *uuid.UUID, total, amount int64) uuid.UUID {
		schedID := uuid.New()
		mustExec(t, conn, `INSERT INTO revenue_schedules (id, tenant_id, invoice_id, entity_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'USD', NOW() - INTERVAL '1 month', NOW() + INTERVAL '11 months', 'active', NOW(), NOW())`,
			schedID, tenantID, invID, entityID, total)
		mustExec(t, conn, `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			VALUES ($1,$2,$3,$4, NOW(), 'pending', NOW())`,
			uuid.New(), schedID, tenantID, amount)
		return schedID
	}

	// Primary entity: Deferred 1000, scheduled 40 (healthy). entity nil ⇒ primary.
	primaryInv := seedFundedEntity(nil, 1000, "primary")
	schedulePending(primaryInv, nil, 1000, 40)

	// Second entity: Deferred 50, scheduled 50 (healthy for phase 1).
	entity2Inv := seedFundedEntity(&entity2, 50, "eu")
	entity2Sched := schedulePending(entity2Inv, &entity2, 50, 50)

	// --- Phase 1: legitimate multi-entity books reconcile clean. ---
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("Run (phase 1): %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyDeferredBelowScheduled {
			t.Fatalf("phase 1 false-positive deferred_below_scheduled on healthy books: %+v", d)
		}
	}

	// --- Phase 2: the second entity develops a per-entity shortfall (scheduled
	// 100 > Deferred 50), masked in the aggregate by the primary's excess. ---
	mustExec(t, conn, `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
		VALUES ($1,$2,$3,50, NOW(), 'pending', NOW())`,
		uuid.New(), entity2Sched, tenantID)

	report, err = recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("Run (phase 2): %v", err)
	}
	var deferredFindings []ReconciliationDiscrepancy
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyDeferredBelowScheduled {
			deferredFindings = append(deferredFindings, d)
		}
	}
	if len(deferredFindings) != 1 {
		t.Fatalf("phase 2: want exactly 1 deferred_below_scheduled (the second entity's), got %d: %+v",
			len(deferredFindings), report.Discrepancies)
	}
	d := deferredFindings[0]
	if d.ExpectedAmount != 100 || d.FoundAmount != 50 {
		t.Errorf("phase 2 finding = scheduled %d / deferred %d, want 100 / 50 (second entity)", d.ExpectedAmount, d.FoundAmount)
	}
}
