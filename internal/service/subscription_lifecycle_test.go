package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// lifecycleSubRepo is a minimal SubscriptionRepository for the pause/resume/
// reactivate paths, which touch only GetByID + Update. GetByID mirrors the real
// repo: a missing row returns (nil, nil) — the case ResumeSubscription must not
// dereference.
type lifecycleSubRepo struct {
	port.SubscriptionRepository
	subs    map[uuid.UUID]*domain.Subscription
	updated int
}

func (r *lifecycleSubRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return r.subs[id], nil
}

func (r *lifecycleSubRepo) Update(_ context.Context, sub *domain.Subscription) error {
	r.subs[sub.ID] = sub
	r.updated++
	return nil
}

func (r *lifecycleSubRepo) SetResumeAt(_ context.Context, _, subID uuid.UUID, resumeAt *time.Time) error {
	if s := r.subs[subID]; s != nil {
		s.ResumeAt = resumeAt
	}
	return nil
}

func newLifecycleSvc(sub *domain.Subscription) (*SubscriptionService, *lifecycleSubRepo) {
	repo := &lifecycleSubRepo{subs: map[uuid.UUID]*domain.Subscription{}}
	if sub != nil {
		repo.subs[sub.ID] = sub
	}
	return &SubscriptionService{subRepo: repo}, repo
}

func activeSub(tenant uuid.UUID) *domain.Subscription {
	return &domain.Subscription{
		ID:               uuid.New(),
		TenantID:         tenant,
		Status:           domain.SubscriptionStatusActive,
		CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}
}

// TestSubscriptionLifecycle_PauseResume covers the happy path and the guards:
// only active can pause, only paused can resume, and the transitions persist.
func TestSubscriptionLifecycle_PauseResume(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	svc, repo := newLifecycleSvc(sub)
	ctx := context.Background()

	// Pause an active subscription.
	if _, err := svc.PauseSubscription(ctx, tenant, sub.ID, nil); err != nil {
		t.Fatalf("pause active: %v", err)
	}
	if repo.subs[sub.ID].Status != domain.SubscriptionStatusPaused {
		t.Fatalf("status after pause = %s, want paused", repo.subs[sub.ID].Status)
	}

	// Pausing again (now paused) is rejected.
	if _, err := svc.PauseSubscription(ctx, tenant, sub.ID, nil); err == nil {
		t.Fatal("pausing a paused subscription should fail")
	}

	// Resume the paused subscription.
	if _, err := svc.ResumeSubscription(ctx, tenant, sub.ID); err != nil {
		t.Fatalf("resume paused: %v", err)
	}
	if repo.subs[sub.ID].Status != domain.SubscriptionStatusActive {
		t.Fatalf("status after resume = %s, want active", repo.subs[sub.ID].Status)
	}

	// Resuming an active subscription is rejected.
	if _, err := svc.ResumeSubscription(ctx, tenant, sub.ID); err == nil {
		t.Fatal("resuming an active subscription should fail")
	}
}

// TestSubscriptionLifecycle_TimedPauseSetsAndClearsResumeAt proves the issue
// #111 wiring: a timed pause records resume_at, and resuming clears it so the
// resume scheduler won't re-claim the row.
func TestSubscriptionLifecycle_TimedPauseSetsAndClearsResumeAt(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	svc, repo := newLifecycleSvc(sub)
	ctx := context.Background()
	resumeAt := time.Now().Add(90 * 24 * time.Hour)

	if _, err := svc.PauseSubscription(ctx, tenant, sub.ID, &resumeAt); err != nil {
		t.Fatalf("timed pause: %v", err)
	}
	if got := repo.subs[sub.ID].ResumeAt; got == nil || !got.Equal(resumeAt) {
		t.Fatalf("ResumeAt after timed pause = %v, want %v", got, resumeAt)
	}

	if _, err := svc.ResumeSubscription(ctx, tenant, sub.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got := repo.subs[sub.ID].ResumeAt; got != nil {
		t.Errorf("ResumeAt after resume = %v, want nil (cleared)", got)
	}
}

// lifecyclePlanRepo returns a fixed monthly plan for the resume period-roll.
type lifecyclePlanRepo struct {
	port.PlanRepository
	plan *domain.Plan
}

func (r *lifecyclePlanRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Plan, error) {
	return r.plan, nil
}

// TestSubscriptionLifecycle_ResumeRollsElapsedPeriodForward proves the
// back-billing guard: resuming a subscription whose period fully elapsed during
// the pause rolls the period forward from now, so the renewal scheduler doesn't
// retroactively bill the paused window.
func TestSubscriptionLifecycle_ResumeRollsElapsedPeriodForward(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	sub.Status = domain.SubscriptionStatusPaused
	sub.PlanID = uuid.New()
	// Period ended two months ago (a long pause).
	sub.CurrentPeriodStart = time.Now().AddDate(0, -3, 0)
	sub.CurrentPeriodEnd = time.Now().AddDate(0, -2, 0)

	repo := &lifecycleSubRepo{subs: map[uuid.UUID]*domain.Subscription{sub.ID: sub}}
	planRepo := &lifecyclePlanRepo{plan: &domain.Plan{ID: sub.PlanID, IntervalUnit: domain.IntervalMonth, IntervalCount: 1}}
	svc := &SubscriptionService{subRepo: repo, planRepo: planRepo}

	if _, err := svc.ResumeSubscription(context.Background(), tenant, sub.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	got := repo.subs[sub.ID]
	if got.Status != domain.SubscriptionStatusActive {
		t.Fatalf("status = %s, want active", got.Status)
	}
	// Period must now be in the future (fresh from ~now), not the stale past — else
	// the renewal scheduler would back-bill the paused months.
	if !got.CurrentPeriodEnd.After(time.Now()) {
		t.Errorf("current_period_end = %v, want a future date (rolled forward)", got.CurrentPeriodEnd)
	}
	if got.CurrentPeriodStart.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("current_period_start = %v, want ~now", got.CurrentPeriodStart)
	}
}

// A brief pause that ends within the original period keeps its remaining time
// (no roll-forward, no plan lookup needed).
func TestSubscriptionLifecycle_ResumeKeepsUnexpiredPeriod(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant) // CurrentPeriodEnd = now + 30d
	sub.Status = domain.SubscriptionStatusPaused
	origEnd := sub.CurrentPeriodEnd
	// No planRepo wired: if the code tried to roll forward it would nil-panic,
	// proving the unexpired path doesn't touch the plan.
	repo := &lifecycleSubRepo{subs: map[uuid.UUID]*domain.Subscription{sub.ID: sub}}
	svc := &SubscriptionService{subRepo: repo}

	if _, err := svc.ResumeSubscription(context.Background(), tenant, sub.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if !repo.subs[sub.ID].CurrentPeriodEnd.Equal(origEnd) {
		t.Errorf("current_period_end changed to %v, want unchanged %v", repo.subs[sub.ID].CurrentPeriodEnd, origEnd)
	}
}

// TestSubscriptionLifecycle_NotFoundAndCrossTenant proves the guards fail
// closed: a missing subscription (GetByID -> nil) does not panic and a
// cross-tenant caller is refused — for pause AND resume (resume previously
// lacked the nil check).
func TestSubscriptionLifecycle_NotFoundAndCrossTenant(t *testing.T) {
	tenant := uuid.New()
	other := uuid.New()
	ctx := context.Background()

	// Missing subscription: both must return an error, not panic.
	svc, _ := newLifecycleSvc(nil)
	missing := uuid.New()
	if _, err := svc.PauseSubscription(ctx, tenant, missing, nil); err == nil {
		t.Fatal("pause missing: expected error")
	}
	if _, err := svc.ResumeSubscription(ctx, tenant, missing); err == nil {
		t.Fatal("resume missing: expected error (and must not nil-deref)")
	}

	// Cross-tenant: a paused sub owned by `other` is invisible to `tenant`.
	paused := activeSub(other)
	paused.Status = domain.SubscriptionStatusPaused
	svc, repo := newLifecycleSvc(paused)
	if _, err := svc.ResumeSubscription(ctx, tenant, paused.ID); err == nil {
		t.Fatal("cross-tenant resume: expected error")
	}
	if repo.updated != 0 {
		t.Fatalf("cross-tenant resume mutated state (%d updates)", repo.updated)
	}
}

// TestSubscriptionLifecycle_CancelIdempotent proves the ENG-183 guard: canceling
// an already-canceled subscription is a no-op — it returns the terminal state
// without re-writing (which would reset CanceledAt and re-run the gateway
// cancel + rev-rec unwind).
func TestSubscriptionLifecycle_CancelIdempotent(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	sub.Status = domain.SubscriptionStatusCanceled
	now := time.Now()
	sub.CanceledAt = &now
	svc, repo := newLifecycleSvc(sub)

	res, err := svc.Cancel(context.Background(), tenant, sub.ID, true, "", "")
	if err != nil {
		t.Fatalf("idempotent cancel: %v", err)
	}
	if res == nil || res.Status != string(domain.SubscriptionStatusCanceled) {
		t.Fatalf("idempotent cancel result = %+v, want canceled", res)
	}
	if repo.updated != 0 {
		t.Errorf("already-canceled Cancel wrote to the repo %d times, want 0", repo.updated)
	}
}

// TestSubscriptionLifecycle_ReactivateClearsCanceledAt proves the ENG-183 fix:
// reactivating clears CanceledAt, so downstream churn/MRR/rev-rec queries that
// filter canceled_at IS NOT NULL don't misclassify the live subscription.
func TestSubscriptionLifecycle_ReactivateClearsCanceledAt(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	sub.Status = domain.SubscriptionStatusCanceled
	sub.CancelAtPeriodEnd = true
	now := time.Now()
	sub.CanceledAt = &now
	svc, repo := newLifecycleSvc(sub)

	if _, err := svc.Reactivate(context.Background(), tenant, sub.ID); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if repo.subs[sub.ID].CanceledAt != nil {
		t.Errorf("Reactivate left CanceledAt = %v, want nil", repo.subs[sub.ID].CanceledAt)
	}
	if repo.subs[sub.ID].Status != domain.SubscriptionStatusActive {
		t.Errorf("status after reactivate = %s, want active", repo.subs[sub.ID].Status)
	}
}

// TestSubscriptionLifecycle_Reactivate covers reactivating a cancel-at-period-end
// subscription and the period-ended guard.
func TestSubscriptionLifecycle_Reactivate(t *testing.T) {
	tenant := uuid.New()
	ctx := context.Background()

	// cancel_at_period_end within the period -> reactivates to active.
	sub := activeSub(tenant)
	sub.CancelAtPeriodEnd = true
	svc, repo := newLifecycleSvc(sub)
	if _, err := svc.Reactivate(ctx, tenant, sub.ID); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if repo.subs[sub.ID].Status != domain.SubscriptionStatusActive || repo.subs[sub.ID].CancelAtPeriodEnd {
		t.Fatalf("after reactivate: status=%s cape=%v, want active/false",
			repo.subs[sub.ID].Status, repo.subs[sub.ID].CancelAtPeriodEnd)
	}

	// Period already ended -> cannot reactivate.
	expired := activeSub(tenant)
	expired.CancelAtPeriodEnd = true
	expired.CurrentPeriodEnd = time.Now().Add(-time.Hour)
	svc, _ = newLifecycleSvc(expired)
	if _, err := svc.Reactivate(ctx, tenant, expired.ID); err == nil {
		t.Fatal("reactivating after period end should fail")
	}
}
