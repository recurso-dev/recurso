package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/adapter/email"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
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

	log.Println("✅ Pre-charge scheduler started (runs hourly)")
}

// Stop stops the scheduler
func (s *PreChargeScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("🛑 Pre-charge scheduler stopped")
}

// runPreChargeNotifications sends notifications for subscriptions due tomorrow
func (s *PreChargeScheduler) runPreChargeNotifications() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:pre-charge"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to obtain lock for pre-charge scheduler: %v", err)
		return
	}
	if !acquired {
		// Lock is held by another instance, skip this run
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			log.Printf("Failed to release lock for pre-charge scheduler: %v", err)
		}
	}()

	subscriptions, err := s.subscriptionRepo.GetSubscriptionsDueTomorrow(ctx)
	if err != nil {
		log.Printf("Error fetching subscriptions for pre-charge: %v", err)
		return
	}

	if len(subscriptions) == 0 {
		log.Println("No subscriptions due tomorrow for pre-charge notification")
		return
	}

	log.Printf("Found %d subscriptions due tomorrow for pre-charge notification", len(subscriptions))

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
			log.Printf("Failed to send pre-charge notification for subscription %s: %v", sub.ID, err)
			continue
		}

		// Mark as sent
		if err := s.subscriptionRepo.MarkPreChargeNotificationSent(ctx, sub.ID, sub.NextBillingDate); err != nil {
			log.Printf("Failed to mark pre-charge notification as sent for subscription %s: %v", sub.ID, err)
		}

		log.Printf("✉️  Sent pre-charge notification for subscription %s to %s", sub.ID, sub.CustomerEmail)
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
