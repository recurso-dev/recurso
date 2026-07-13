package db

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestActivateTrialWithTx_ConcurrentTransitionsElectOneWinner proves the ENG-161
// trial-conversion fix: when several runners try to convert the same expired
// trial at once (as multi-instance schedulers do when the distributed lock is a
// no-op), the conditional `... WHERE status='trialing'` transition is won by
// EXACTLY ONE runner. Only that winner goes on to create the first invoice, so a
// trial is billed once instead of once-per-instance.
func TestActivateTrialWithTx_ConcurrentTransitionsElectOneWinner(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed activate-trial test")
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
		tenantID, "Trial-"+run, "trial-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, trial_end, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'trialing', NOW() - INTERVAL '14 days', NOW(), NOW() - INTERVAL '14 days', NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		subID, tenantID, customerID, planID); err != nil {
		t.Fatalf("seed trialing subscription: %v", err)
	}

	repo := &SubscriptionRepository{db: conn}

	// Each runner uses its own transaction, exactly like ConvertTrialToActive's
	// txManager.WithTx: begin, conditionally activate, commit on win / roll back
	// on loss.
	now := time.Now().UTC()
	newSub := func() *domain.Subscription {
		return &domain.Subscription{
			ID:                 subID,
			TenantID:           tenantID,
			Status:             domain.SubscriptionStatusActive,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   now.AddDate(0, 1, 0),
			UpdatedAt:          now,
		}
	}

	const runners = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	var mu sync.Mutex
	wins := 0

	for i := 0; i < runners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			tx, err := conn.BeginTx(ctx, nil)
			if err != nil {
				t.Errorf("begin tx: %v", err)
				return
			}
			won, err := repo.ActivateTrialWithTx(ctx, tx, newSub())
			if err != nil {
				_ = tx.Rollback()
				t.Errorf("ActivateTrialWithTx: %v", err)
				return
			}
			if won {
				if err := tx.Commit(); err != nil {
					t.Errorf("commit: %v", err)
					return
				}
				mu.Lock()
				wins++
				mu.Unlock()
			} else {
				_ = tx.Rollback()
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("trial was activated by %d runners, want exactly 1 (double-billing race)", wins)
	}

	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM subscriptions WHERE id = $1`, subID).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != string(domain.SubscriptionStatusActive) {
		t.Errorf("final status = %q, want active", status)
	}

	// A follow-up transition finds nothing to do — the trial is no longer
	// trialing, so it can never be converted (billed) a second time.
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	won, err := repo.ActivateTrialWithTx(ctx, tx, newSub())
	_ = tx.Rollback()
	if err != nil {
		t.Fatalf("second ActivateTrialWithTx: %v", err)
	}
	if won {
		t.Error("already-active subscription was re-converted — would double-bill")
	}
}
