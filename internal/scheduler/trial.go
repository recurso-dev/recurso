package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// defaultTrialReminderWindow is how far before trial_end the "trial ending"
// reminder is sent.
const defaultTrialReminderWindow = 3 * 24 * time.Hour

// TrialSubscriptionRepo is the narrow repository surface the trial scheduler
// needs (implemented by *db.SubscriptionRepository).
type TrialSubscriptionRepo interface {
	GetExpiredTrials(ctx context.Context) ([]*domain.Subscription, error)
	GetTrialsEndingWithin(ctx context.Context, within time.Duration) ([]db.TrialEndingNotice, error)
	MarkTrialReminderSent(ctx context.Context, subscriptionID uuid.UUID) error
}

// TrialConverter converts an expired trial to an active subscription and
// generates its first invoice (implemented by *service.SubscriptionService).
type TrialConverter interface {
	ConvertTrialToActive(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error)
}

// TrialNotifier sends trial-ending reminders (implemented by
// *service.NotificationService).
type TrialNotifier interface {
	SendTrialEndingReminder(ctx context.Context, data email.TrialEndingEmailData) error
}

// TrialScheduler drives the trial lifecycle: it sends a trial-ending reminder
// before expiry and, at trial_end, converts trialing subscriptions to active
// (generating the first real invoice, which then flows into the normal
// payment/dunning path).
type TrialScheduler struct {
	repo           TrialSubscriptionRepo
	converter      TrialConverter
	notifier       TrialNotifier
	locker         port.Locker
	portalBaseURL  string
	reminderWindow time.Duration
	ticker         *time.Ticker
	done           chan bool
}

// NewTrialScheduler creates a trial scheduler with the default 3-day reminder window.
func NewTrialScheduler(
	repo TrialSubscriptionRepo,
	converter TrialConverter,
	notifier TrialNotifier,
	locker port.Locker,
	portalBaseURL string,
) *TrialScheduler {
	return &TrialScheduler{
		repo:           repo,
		converter:      converter,
		notifier:       notifier,
		locker:         locker,
		portalBaseURL:  portalBaseURL,
		reminderWindow: defaultTrialReminderWindow,
		done:           make(chan bool),
	}
}

// Start begins the trial scheduler (runs every 6 hours).
func (s *TrialScheduler) Start() {
	s.ticker = time.NewTicker(6 * time.Hour)

	// Run immediately on start.
	go s.processTrials()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.processTrials()
			}
		}
	}()

	slog.Info("trial scheduler started (runs every 6 hours)")
}

// Stop stops the scheduler.
func (s *TrialScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	slog.Info("trial scheduler stopped")
}

// processTrials sends reminders then converts expired trials, under a
// distributed lock so only one instance runs the job.
func (s *TrialScheduler) processTrials() {
	ctx := context.Background()

	lockKey := "scheduler:trials"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 15*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for trial scheduler", "error", err)
		return
	}
	if !acquired {
		return // Lock held by another instance
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for trial scheduler", "error", err)
		}
	}()

	s.sendReminders(ctx)
	s.convertExpiredTrials(ctx)
}

// sendReminders emails customers whose trial ends within the reminder window,
// marking each so it is sent at most once.
func (s *TrialScheduler) sendReminders(ctx context.Context) {
	notices, err := s.repo.GetTrialsEndingWithin(ctx, s.reminderWindow)
	if err != nil {
		slog.Error("failed to fetch trials ending soon", "error", err)
		return
	}

	for _, n := range notices {
		data := email.TrialEndingEmailData{
			CustomerName:  n.CustomerName,
			CustomerEmail: n.CustomerEmail,
			PlanName:      n.PlanName,
			Amount:        formatAmount(n.Amount, n.Currency),
			TrialEndDate:  n.TrialEnd.Format("January 2, 2006"),
			// The SPA's portal entry is /portal/login — bare /portal matches
			// nothing and the router catch-all bounces to the merchant app.
			PortalURL: s.portalBaseURL + "/portal/login",
		}

		if err := s.notifier.SendTrialEndingReminder(ctx, data); err != nil {
			slog.Error("failed to send trial-ending reminder", "subscription_id", n.SubscriptionID, "error", err)
			continue
		}

		if err := s.repo.MarkTrialReminderSent(ctx, n.SubscriptionID); err != nil {
			slog.Error("failed to mark trial reminder sent", "subscription_id", n.SubscriptionID, "error", err)
		} else {
			slog.Info("sent trial-ending reminder", "subscription_id", n.SubscriptionID, "customer_email", n.CustomerEmail)
		}
	}
}

// convertExpiredTrials converts every trialing subscription whose trial_end has
// passed to active, generating the first invoice.
func (s *TrialScheduler) convertExpiredTrials(ctx context.Context) {
	subs, err := s.repo.GetExpiredTrials(ctx)
	if err != nil {
		slog.Error("failed to fetch expired trials", "error", err)
		return
	}

	for _, sub := range subs {
		inv, err := s.converter.ConvertTrialToActive(ctx, sub)
		if err != nil {
			// Another runner already converted this trial (multi-instance race) —
			// benign, not a failure.
			if errors.Is(err, service.ErrTrialAlreadyConverted) {
				continue
			}
			slog.Error("failed to convert trial for subscription", "subscription_id", sub.ID, "error", err)
			continue
		}
		slog.Info("converted trial to active", "subscription_id", sub.ID, "invoice_id", inv.ID)
	}
}
