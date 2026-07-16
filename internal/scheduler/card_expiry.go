package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// CustomerRepoForCardExpiry is a narrow interface for the scheduler
type CustomerRepoForCardExpiry interface {
	GetCustomersWithExpiringCards(ctx context.Context, month, year int) ([]db.CustomerWithExpiringCard, error)
	MarkCardExpiryNotificationSent(ctx context.Context, customerID, tenantID uuid.UUID, expMonth, expYear int, cardLast4 string) error
}

// CardExpiringScheduler sends notifications for cards expiring soon
type CardExpiringScheduler struct {
	customerRepo    CustomerRepoForCardExpiry
	notificationSvc *service.NotificationService
	locker          port.Locker
	portalBaseURL   string
	ticker          *time.Ticker
	done            chan bool
	stopOnce        sync.Once
}

// NewCardExpiringScheduler creates a new card expiry scheduler
func NewCardExpiringScheduler(
	customerRepo CustomerRepoForCardExpiry,
	notificationSvc *service.NotificationService,
	locker port.Locker,
	portalBaseURL string,
) *CardExpiringScheduler {
	return &CardExpiringScheduler{
		customerRepo:    customerRepo,
		notificationSvc: notificationSvc,
		locker:          locker,
		portalBaseURL:   portalBaseURL,
		done:            make(chan bool),
	}
}

// Start begins the scheduler (runs every 12 hours)
func (s *CardExpiringScheduler) Start() {
	s.ticker = time.NewTicker(12 * time.Hour)

	// Run immediately on start
	go s.runCardExpiryNotifications()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runCardExpiryNotifications()
			}
		}
	}()

	slog.Info("card expiry scheduler started (runs every 12 hours)")
}

// Stop stops the scheduler
func (s *CardExpiringScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("card expiry scheduler stopped")
	})
}

// runCardExpiryNotifications sends notifications for cards expiring next month
func (s *CardExpiringScheduler) runCardExpiryNotifications() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:card-expiry"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for card expiry scheduler", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for card expiry scheduler", "error", err)
		}
	}()

	// Calculate target: cards expiring in the next month
	target := time.Now().AddDate(0, 1, 0)
	targetMonth := int(target.Month())
	targetYear := target.Year()

	customers, err := s.customerRepo.GetCustomersWithExpiringCards(ctx, targetMonth, targetYear)
	if err != nil {
		slog.Error("failed to fetch customers with expiring cards", "error", err)
		return
	}

	if len(customers) == 0 {
		slog.Info("no cards expiring next month requiring notification")
		return
	}

	slog.Info("found customers with expiring cards", "count", len(customers), "expiry_month", targetMonth, "expiry_year", targetYear)

	for _, cust := range customers {
		expiryDate := fmt.Sprintf("%s %d", time.Month(cust.CardExpMonth).String(), cust.CardExpYear)

		data := email.CardExpiringEmailData{
			CustomerName:     cust.CustomerName,
			CustomerEmail:    cust.CustomerEmail,
			CardBrand:        cust.CardBrand,
			CardLast4:        cust.CardLast4,
			ExpiryDate:       expiryDate,
			UpdatePaymentURL: s.portalBaseURL + "/portal/login",
		}

		if err := s.notificationSvc.SendCardExpiringNotification(ctx, data); err != nil {
			slog.Error("failed to send card expiry notification", "customer_id", cust.CustomerID, "error", err)
			continue
		}

		if err := s.customerRepo.MarkCardExpiryNotificationSent(ctx, cust.CustomerID, cust.TenantID, cust.CardExpMonth, cust.CardExpYear, cust.CardLast4); err != nil {
			slog.Error("failed to mark card expiry notification as sent", "customer_id", cust.CustomerID, "error", err)
		}

		slog.Info("sent card expiry notification", "customer_id", cust.CustomerID, "customer_email", cust.CustomerEmail)
	}
}
