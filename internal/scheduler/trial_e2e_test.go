package scheduler

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/email"
	"github.com/swapnull-in/recur-so/internal/adapter/memory"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

type e2eTrialNotifier struct {
	reminders []email.TrialEndingEmailData
}

func (n *e2eTrialNotifier) SendTrialEndingReminder(ctx context.Context, data email.TrialEndingEmailData) error {
	n.reminders = append(n.reminders, data)
	return nil
}

// TestTrialFlow_EndToEnd_Postgres covers ENG-3 (FR-BIL-8) against real SQL:
// a subscription created with trial days starts trialing with NO first
// invoice; the reminder fires (once) inside the reminder window with a
// working portal link; at expiry the scheduler converts it to active with a
// correct first invoice whose period starts where the trial ended; and that
// invoice is visible to the dunning path (GetOverdueInvoices), not silent
// limbo.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database, e.g.:
//
//	createdb recurso_repo_test
//	TEST_DATABASE_URL='postgres://localhost:5432/recurso_repo_test?sslmode=disable' go test ./internal/scheduler/
func TestTrialFlow_EndToEnd_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed trial e2e")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	sqlxConn := sqlx.NewDb(conn, "postgres")
	ctx := context.Background()

	// --- Seed tenant, USD plan (tax-free for a US buyer), customer ---
	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Trial-E2E-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	planRepo := db.NewPlanRepository(conn)
	planID := uuid.New()
	plan := &domain.Plan{
		ID: planID, TenantID: tenantID, Name: "Trial Pro", Code: "trial-pro-" + tenantID.String()[:8],
		IntervalUnit: domain.IntervalUnit("month"), IntervalCount: 1, Active: true, CreatedAt: time.Now(),
		Prices: []domain.Price{{ID: uuid.New(), PlanID: planID, Currency: "USD", Amount: 2900, Type: "recurring", CreatedAt: time.Now()}},
	}
	if err := planRepo.Create(tenantCtx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	custRepo := db.NewCustomerRepository(sqlxConn)
	name := "Trial Customer"
	customer := &domain.Customer{
		ID: uuid.New(), TenantID: tenantID, Name: &name,
		Email:          "trial-" + tenantID.String()[:8] + "@example.com",
		BillingAddress: domain.BillingAddress{Line1: "1 Market St", City: "SF", State: "CA", Zip: "94105", Country: "US"},
		CreatedAt:      time.Now(), UpdatedAt: time.Now(),
	}
	if err := custRepo.Create(tenantCtx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	subRepoPort := db.NewSubscriptionRepository(conn)
	subRepo, ok := subRepoPort.(*db.SubscriptionRepository)
	if !ok {
		t.Fatalf("subscription repo is %T, want *db.SubscriptionRepository", subRepoPort)
	}
	invRepo := db.NewInvoiceRepository(conn)
	svc := service.NewSubscriptionService(
		subRepoPort, invRepo, planRepo, custRepo, db.NewCouponRepository(conn),
		nil, nil, nil, nil, db.NewTxManager(conn), nil, nil,
	)

	// --- (pre-a) Trial creation: trialing, trial_end set, NO first invoice ---
	sub, err := svc.CreateSubscription(tenantCtx, service.CreateSubscriptionInput{
		TenantID: tenantID, CustomerID: customer.ID, PlanID: plan.ID, TrialDays: 7,
	})
	if err != nil {
		t.Fatalf("create trial subscription: %v", err)
	}
	if sub.Status != domain.SubscriptionStatusTrialing {
		t.Fatalf("status = %s, want trialing", sub.Status)
	}
	if sub.TrialEnd == nil {
		t.Fatal("trial_end not set")
	}
	if invs, _ := invRepo.GetByCustomerID(ctx, customer.ID); len(invs) != 0 {
		t.Fatalf("trial creation generated %d invoices, want 0", len(invs))
	}

	// --- Force the trial into the reminder window, run the scheduler tick ---
	trialEnd := time.Now().UTC().Add(48 * time.Hour)
	if _, err := conn.ExecContext(ctx, `UPDATE subscriptions SET trial_end = $1 WHERE id = $2`, trialEnd, sub.ID); err != nil {
		t.Fatalf("backdate to reminder window: %v", err)
	}
	notifier := &e2eTrialNotifier{}
	sched := NewTrialScheduler(subRepo, svc, notifier, memory.NewNoOpLocker(), "https://billing.test")
	sched.processTrials()

	// (b) Reminder fired with a working portal link, and only once.
	if len(notifier.reminders) != 1 {
		t.Fatalf("reminders = %d, want 1", len(notifier.reminders))
	}
	if got := notifier.reminders[0].PortalURL; got != "https://billing.test/portal/login" {
		t.Errorf("reminder portal URL = %q, want /portal/login (bare /portal is a dead route)", got)
	}
	sched.processTrials()
	if len(notifier.reminders) != 1 {
		t.Fatalf("reminder re-sent on second tick: %d total", len(notifier.reminders))
	}

	// --- (a) Expire the trial; the tick converts it with a correct invoice ---
	expiredEnd := time.Now().UTC().Add(-1 * time.Hour)
	if _, err := conn.ExecContext(ctx, `UPDATE subscriptions SET trial_end = $1 WHERE id = $2`, expiredEnd, sub.ID); err != nil {
		t.Fatalf("expire trial: %v", err)
	}
	sched.processTrials()

	converted, err := subRepo.GetByID(tenantCtx, sub.ID)
	if err != nil || converted == nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if converted.Status != domain.SubscriptionStatusActive {
		t.Fatalf("status after conversion = %s, want active (tenant-context regression?)", converted.Status)
	}
	if !converted.CurrentPeriodStart.Equal(expiredEnd.Truncate(time.Microsecond)) &&
		converted.CurrentPeriodStart.Sub(expiredEnd).Abs() > time.Second {
		t.Errorf("first paid period starts %v, want the trial end %v", converted.CurrentPeriodStart, expiredEnd)
	}

	invs, err := invRepo.GetByCustomerID(ctx, customer.ID)
	if err != nil || len(invs) != 1 {
		t.Fatalf("invoices after conversion = %d (err=%v), want exactly 1", len(invs), err)
	}
	inv := invs[0]
	if inv.Status != domain.InvoiceStatusOpen {
		t.Errorf("first invoice status = %s, want open", inv.Status)
	}
	if inv.Subtotal != 2900 || inv.Total != 2900 {
		t.Errorf("first invoice subtotal/total = %d/%d, want 2900/2900 (US buyer, no tax)", inv.Subtotal, inv.Total)
	}

	// A second tick must not convert again or double-invoice (idempotent).
	sched.processTrials()
	if invs, _ := invRepo.GetByCustomerID(ctx, customer.ID); len(invs) != 1 {
		t.Fatalf("second tick created another invoice: %d total", len(invs))
	}

	// --- (c) The unpaid first invoice enters the dunning path, not limbo ---
	if _, err := conn.ExecContext(ctx, `UPDATE invoices SET due_date = NOW() - INTERVAL '1 hour' WHERE id = $1`, inv.ID); err != nil {
		t.Fatalf("backdate due date: %v", err)
	}
	invRepoImpl, ok := invRepo.(*db.InvoiceRepository)
	if !ok {
		t.Fatalf("invoice repo is %T", invRepo)
	}
	overdue, err := invRepoImpl.GetOverdueInvoices(ctx)
	if err != nil {
		t.Fatalf("GetOverdueInvoices: %v", err)
	}
	found := false
	for _, o := range overdue {
		if strings.EqualFold(o.ID.String(), inv.ID.String()) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("trial-conversion invoice %s not picked up by GetOverdueInvoices — failed first payments would sit in silent limbo", inv.ID)
	}
}
