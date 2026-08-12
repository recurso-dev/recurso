package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestListFilters_Postgres proves the server-side list filters added for the
// dashboard's honest-filter wave (#595) against real Postgres — mock repo
// tests give zero coverage of the actual WHERE clauses:
//   - SubscriptionFilter.PlanID and .StartedAfter (current_period_start bound)
//   - PlanFilter.Currency (EXISTS over the plan's prices) and .IntervalUnit
//   - EventRepository.ListByTenantID's eventType filter
//
// Each filter is also checked tenant-scoped / non-matching → empty.
func TestListFilters_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed list-filter test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Filters-"+run, "filters-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	// Two plans: monthly with a USD price, yearly with an INR price.
	planUSD, planINR := uuid.New(), uuid.New()
	seedPlan := func(id uuid.UUID, code, interval, currency string) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, $3, $4, $5, 1, TRUE)`,
			id, tenantID, code, code+"-"+run, interval); err != nil {
			t.Fatalf("seed plan %s: %v", code, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1, $2, $3, 1000, 'recurring')`,
			uuid.New(), id, currency); err != nil {
			t.Fatalf("seed price %s: %v", code, err)
		}
	}
	seedPlan(planUSD, "monthly", "month", "USD")
	seedPlan(planINR, "yearly", "year", "INR")

	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		custID, tenantID, "c-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// Two subscriptions: one on each plan; the USD one's period started long
	// ago, the INR one's period started now.
	subOld, subNew := uuid.New(), uuid.New()
	seedSub := func(id, planID uuid.UUID, periodStart time.Time) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'active', $5::timestamptz, $5::timestamptz + INTERVAL '1 month', $5::timestamptz, NOW(), NOW())`,
			id, tenantID, custID, planID, periodStart); err != nil {
			t.Fatalf("seed subscription: %v", err)
		}
	}
	now := time.Now().UTC()
	seedSub(subOld, planUSD, now.AddDate(0, -3, 0))
	seedSub(subNew, planINR, now)

	subRepo := &SubscriptionRepository{db: conn}

	// PlanID filter: only the USD plan's subscription.
	got, err := subRepo.List(ctx, tenantID, domain.SubscriptionFilter{PlanID: planUSD})
	if err != nil {
		t.Fatalf("List(plan): %v", err)
	}
	if len(got) != 1 || got[0].ID != subOld {
		t.Errorf("plan filter returned %d subs (want 1: the USD plan's)", len(got))
	}

	// StartedAfter: a bound between the two period starts keeps only the new one.
	after := now.AddDate(0, -1, 0)
	got, err = subRepo.List(ctx, tenantID, domain.SubscriptionFilter{StartedAfter: &after})
	if err != nil {
		t.Fatalf("List(started_after): %v", err)
	}
	if len(got) != 1 || got[0].ID != subNew {
		t.Errorf("started_after filter returned %d subs (want 1: the recent one)", len(got))
	}

	// Combined: plan + date that excludes that plan's only subscription → empty.
	got, err = subRepo.List(ctx, tenantID, domain.SubscriptionFilter{PlanID: planUSD, StartedAfter: &after})
	if err != nil {
		t.Fatalf("List(plan+started_after): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("plan+date filter returned %d subs, want 0", len(got))
	}

	planRepo := NewPlanRepository(conn)

	// Currency filter goes through the prices EXISTS.
	plans, err := planRepo.List(ctx, tenantID, domain.PlanFilter{Currency: "INR"})
	if err != nil {
		t.Fatalf("List(currency): %v", err)
	}
	if len(plans) != 1 || plans[0].ID != planINR {
		t.Errorf("currency filter returned %d plans (want 1: the INR-priced one)", len(plans))
	}

	// IntervalUnit filter.
	plans, err = planRepo.List(ctx, tenantID, domain.PlanFilter{IntervalUnit: "month"})
	if err != nil {
		t.Fatalf("List(interval): %v", err)
	}
	if len(plans) != 1 || plans[0].ID != planUSD {
		t.Errorf("interval filter returned %d plans (want 1: the monthly one)", len(plans))
	}

	// A currency no plan prices in → empty, not an error.
	plans, err = planRepo.List(ctx, tenantID, domain.PlanFilter{Currency: "JPY"})
	if err != nil {
		t.Fatalf("List(currency miss): %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("JPY filter returned %d plans, want 0", len(plans))
	}

	// Events: two types; the filter keeps one, empty string keeps all.
	eventRepo := NewEventRepository(conn)
	seedEvent := func(evType string) {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO events (id, tenant_id, type, object_type, object_id, data) VALUES ($1, $2, $3, 'invoice', $4, '{}')`,
			uuid.New(), tenantID, evType, uuid.New()); err != nil {
			t.Fatalf("seed event: %v", err)
		}
	}
	seedEvent("invoice.paid")
	seedEvent("invoice.paid")
	seedEvent("customer.created")

	events, err := eventRepo.ListByTenantID(ctx, tenantID, "invoice.paid", 50, 0)
	if err != nil {
		t.Fatalf("ListByTenantID(type): %v", err)
	}
	if len(events) != 2 {
		t.Errorf("type filter returned %d events, want 2", len(events))
	}
	for _, e := range events {
		if e.Type != "invoice.paid" {
			t.Errorf("type filter leaked event of type %q", e.Type)
		}
	}
	events, err = eventRepo.ListByTenantID(ctx, tenantID, "", 50, 0)
	if err != nil {
		t.Fatalf("ListByTenantID(all): %v", err)
	}
	if len(events) != 3 {
		t.Errorf("unfiltered list returned %d events, want 3", len(events))
	}
}
