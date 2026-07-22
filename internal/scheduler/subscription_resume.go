package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// resumeClaimWindow is how far ClaimDueForResume pushes resume_at when it claims
// a due subscription. It is the failure lease: long enough that a subscription
// being resumed is never re-claimed mid-flight, and shorter than the daily tick
// so a run that dies leaves the row to be retried on a later tick.
const resumeClaimWindow = 1 * time.Hour

// resumeBatchLimit caps how many subscriptions one tick resumes; successive
// ticks drain any backlog.
const resumeBatchLimit = 100

// pausedSubscriptionClaimer atomically leases paused subscriptions whose
// scheduled resume_at has elapsed. *db.SubscriptionRepository satisfies it.
type pausedSubscriptionClaimer interface {
	ClaimDueForResume(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error)
}

// subscriptionResumer returns a paused subscription to active and clears its
// scheduled resume. *service.SubscriptionService satisfies it.
type subscriptionResumer interface {
	ResumeSubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Subscription, error)
}

// SubscriptionResumeScheduler auto-resumes paused subscriptions whose scheduled
// resume time has passed (issue #111) — e.g. a retention "pause N months" offer.
// Runs daily; the claim (ADR-003) makes it multi-instance-safe even when the
// distributed lock is a no-op (no Redis).
type SubscriptionResumeScheduler struct {
	claimer  pausedSubscriptionClaimer
	resumer  subscriptionResumer
	locker   port.Locker
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

func NewSubscriptionResumeScheduler(
	claimer pausedSubscriptionClaimer,
	resumer subscriptionResumer,
	locker port.Locker,
) *SubscriptionResumeScheduler {
	return &SubscriptionResumeScheduler{
		claimer: claimer,
		resumer: resumer,
		locker:  locker,
		done:    make(chan bool),
	}
}

func (s *SubscriptionResumeScheduler) Start() {
	s.ticker = time.NewTicker(24 * time.Hour)

	// Run once at boot so a pause that elapsed while the service was down is
	// picked up promptly rather than waiting a full day.
	go s.runResumes()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runResumes()
			}
		}
	}()

	slog.Info("subscription resume scheduler started (runs daily)")
}

func (s *SubscriptionResumeScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("subscription resume scheduler stopped")
	})
}

func (s *SubscriptionResumeScheduler) runResumes() {
	ctx := context.Background()

	lockKey := "scheduler:subscription-resume"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for subscription resume scheduler", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for subscription resume scheduler", "error", err)
		}
	}()

	// Atomically CLAIM due paused subscriptions: the lock above is a no-op
	// without Redis, so the claim is what guarantees each is resumed by exactly
	// one runner (ADR-003).
	subs, err := s.claimer.ClaimDueForResume(ctx, resumeClaimWindow, resumeBatchLimit)
	if err != nil {
		slog.Error("failed to claim subscriptions due for resume", "error", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	slog.Info("subscriptions due for auto-resume", "count", len(subs))

	for _, sub := range subs {
		// ResumeSubscription reads through the tenant-scoped repository, so inject
		// the subscription's own tenant (the scheduler ctx carries none). It sets
		// the status active and clears resume_at, so a resumed row is not
		// re-claimed on the next tick.
		tenantCtx := context.WithValue(ctx, domain.TenantIDKey, sub.TenantID)
		if _, err := s.resumer.ResumeSubscription(tenantCtx, sub.TenantID, sub.ID); err != nil {
			slog.Error("failed to auto-resume subscription", "subscription_id", sub.ID, "error", err)
			continue
		}
		slog.Info("auto-resumed subscription", "subscription_id", sub.ID)
	}
}
