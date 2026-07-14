package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// PreChargeScheduler handles 24-hour pre-charge notifications
type PreChargeScheduler struct {
	subscriptionRepo SubscriptionRepoForPreCharge
	notificationSvc  *service.NotificationService
	locker           port.Locker
	portalBaseURL    string
	ticker           *time.Ticker
	done             chan bool
}

// SubscriptionRepoForPreCharge interface for scheduler
type SubscriptionRepoForPreCharge interface {
	GetSubscriptionsDueTomorrow(ctx context.Context) ([]db.SubscriptionWithCustomer, error)
	MarkPreChargeNotificationSent(ctx context.Context, subscriptionID uuid.UUID, chargeDate string) error
}

// NewPreChargeScheduler creates a new scheduler
func NewPreChargeScheduler(
	subscriptionRepo SubscriptionRepoForPreCharge,
	notificationSvc *service.NotificationService,
	locker port.Locker,
	portalBaseURL string,
) *PreChargeScheduler {
	return &PreChargeScheduler{
		subscriptionRepo: subscriptionRepo,
		notificationSvc:  notificationSvc,
		locker:           locker,
		portalBaseURL:    portalBaseURL,
		done:             make(chan bool),
	}
}

// Start begins the scheduler (runs every hour)
func (s *PreChargeScheduler) Start() {
	s.ticker = time.NewTicker(1 * time.Hour)

	// Run immediately on start
	go s.runPreChargeNotifications()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runPreChargeNotifications()
			}
		}
	}()

	slog.Info("pre-charge scheduler started (runs hourly)")
}

// Stop stops the scheduler
func (s *PreChargeScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	slog.Info("pre-charge scheduler stopped")
}

// runPreChargeNotifications sends notifications for subscriptions due tomorrow
func (s *PreChargeScheduler) runPreChargeNotifications() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:pre-charge"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for pre-charge scheduler", "error", err)
		return
	}
	if !acquired {
		// Lock is held by another instance, skip this run
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for pre-charge scheduler", "error", err)
		}
	}()

	subscriptions, err := s.subscriptionRepo.GetSubscriptionsDueTomorrow(ctx)
	if err != nil {
		slog.Error("failed to fetch subscriptions for pre-charge", "error", err)
		return
	}

	if len(subscriptions) == 0 {
		slog.Info("no subscriptions due tomorrow for pre-charge notification")
		return
	}

	slog.Info("found subscriptions due tomorrow for pre-charge notification", "count", len(subscriptions))

	for _, sub := range subscriptions {
		// Send pre-charge reminder email
		data := email.PreChargeEmailData{
			CustomerName:  sub.CustomerName,
			CustomerEmail: sub.CustomerEmail,
			PlanName:      sub.PlanName,
			Amount:        formatAmount(sub.Amount, sub.Currency),
			ChargeDate:    sub.NextBillingDate,
			PaymentMethod: "•••• " + sub.PaymentMethodLast4,
			PortalURL:     s.portalBaseURL + "/portal",
		}

		if err := s.notificationSvc.SendPreChargeReminder(ctx, data); err != nil {
			slog.Error("failed to send pre-charge notification", "subscription_id", sub.ID, "error", err)
			continue
		}

		// Mark as sent
		if err := s.subscriptionRepo.MarkPreChargeNotificationSent(ctx, sub.ID, sub.NextBillingDate); err != nil {
			slog.Error("failed to mark pre-charge notification as sent", "subscription_id", sub.ID, "error", err)
		}

		slog.Info("sent pre-charge notification", "subscription_id", sub.ID, "customer_email", sub.CustomerEmail)
	}
}

func formatAmount(amountPaise int64, currency string) string {
	amount := float64(amountPaise) / 100
	switch currency {
	case "INR":
		return fmt.Sprintf("₹%.2f", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%s %.2f", currency, amount)
	}
}
