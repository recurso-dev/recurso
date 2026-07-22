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

// TestClaimDueForResume_Postgres proves the issue #111 auto-resume claim: a
// paused subscription whose resume_at has elapsed is returned once, its
// resume_at is leased into the future, a second immediate claim skips it, and a
// paused sub with resume_at in the FUTURE (or NULL) is never claimed.
func TestClaimDueForResume_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed resume-claim test")
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
		tenantID, "Resume-"+run, "resume-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1, $2, 'Pro', $3, 'month', 1, TRUE)`,
		planID, tenantID, "pro-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	custID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		custID, tenantID, custID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	seedPaused := func(resumeExpr string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, resume_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'paused', NOW(), NOW() + INTERVAL '1 month', NOW(), `+resumeExpr+`, NOW(), NOW())`,
			id, tenantID, custID, planID); err != nil {
			t.Fatalf("seed paused sub: %v", err)
		}
		return id
	}
	due := seedPaused("NOW() - INTERVAL '1 hour'") // elapsed -> claimable
	future := seedPaused("NOW() + INTERVAL '10 days'")
	indefinite := seedPaused("NULL")

	repo := &SubscriptionRepository{db: conn}

	claimed, err := repo.ClaimDueForResume(ctx, 30*time.Minute, 10)
	if err != nil {
		t.Fatalf("ClaimDueForResume: %v", err)
	}
	got := map[uuid.UUID]bool{}
	for _, s := range claimed {
		got[s.ID] = true
	}
	if !got[due] {
		t.Error("an elapsed paused subscription must be claimed")
	}
	if got[future] || got[indefinite] {
		t.Error("future-dated or NULL resume_at must not be claimed")
	}

	// The claimed row's resume_at must now be leased into the future.
	ctxT := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	after, err := repo.GetByID(ctxT, due)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.ResumeAt == nil || !after.ResumeAt.After(time.Now()) {
		t.Errorf("resume_at after claim = %v, want a future lease", after.ResumeAt)
	}

	// A second immediate claim must not re-return the leased row.
	again, err := repo.ClaimDueForResume(ctx, 30*time.Minute, 10)
	if err != nil {
		t.Fatalf("second ClaimDueForResume: %v", err)
	}
	for _, s := range again {
		if s.ID == due {
			t.Error("a leased subscription must not be re-claimed within its lease window")
		}
	}

	// SetResumeAt(nil) clears it (what ResumeSubscription does on resume).
	if err := repo.SetResumeAt(ctx, tenantID, due, nil); err != nil {
		t.Fatalf("SetResumeAt(nil): %v", err)
	}
	cleared, err := repo.GetByID(ctxT, due)
	if err != nil {
		t.Fatalf("GetByID after clear: %v", err)
	}
	if cleared.ResumeAt != nil {
		t.Errorf("resume_at after SetResumeAt(nil) = %v, want nil", cleared.ResumeAt)
	}
}
