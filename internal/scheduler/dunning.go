package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/email"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

// DunningConfig holds dunning schedule configuration
type DunningConfig struct {
	RetryDays        []int // Days between retries (e.g., [1, 3, 7])
	SuspendAfterDays int   // Suspend service after N days overdue
	CancelAfterDays  int   // Mark uncollectible after N days
	MaxRetries       int   // Maximum retry attempts
}

// DefaultDunningConfig returns sensible defaults
func DefaultDunningConfig() DunningConfig {
	return DunningConfig{
		RetryDays:        []int{2, 3, 4}, // Retry on day 1, 3, 7 after due
		SuspendAfterDays: 7,
		CancelAfterDays:  30,
		MaxRetries:       3,
	}
}

// InvoiceRepoForDunning interface for dunning scheduler
type InvoiceRepoForDunning interface {
	GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error)
	UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error
	UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error
	MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error
}

// DunningScheduler handles payment retry and dunning notifications
type DunningScheduler struct {
	invoiceRepo     InvoiceRepoForDunning
	notificationSvc *service.NotificationService
	locker          port.Locker
	config          DunningConfig
	portalBaseURL   string
	ticker          *time.Ticker
	done            chan bool
}

// NewDunningScheduler creates a new dunning scheduler
func NewDunningScheduler(
	invoiceRepo InvoiceRepoForDunning,
	notificationSvc *service.NotificationService,
	locker port.Locker,
	config DunningConfig,
	portalBaseURL string,
) *DunningScheduler {
	return &DunningScheduler{
		invoiceRepo:     invoiceRepo,
		notificationSvc: notificationSvc,
		locker:          locker,
		config:          config,
		portalBaseURL:   portalBaseURL,
		done:            make(chan bool),
	}
}

// Start begins the dunning scheduler (runs every 6 hours)
func (s *DunningScheduler) Start() {
	s.ticker = time.NewTicker(6 * time.Hour)

	// Run immediately on start
	go s.processDunning()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.processDunning()
			}
		}
	}()

	log.Println("✅ Dunning scheduler started (runs every 6 hours)")
}

// Stop stops the scheduler
func (s *DunningScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("🛑 Dunning scheduler stopped")
}

// processDunning handles all overdue invoices
func (s *DunningScheduler) processDunning() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:dunning"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 30*time.Minute)
	if err != nil {
		log.Printf("Failed to obtain lock for dunning scheduler: %v", err)
		return
	}
	if !acquired {
		return // Lock held by another instance
	}
	defer func() {
		if err := release(ctx); err != nil {
			log.Printf("Failed to release lock for dunning scheduler: %v", err)
		}
	}()

	invoices, err := s.invoiceRepo.GetOverdueInvoices(ctx)
	if err != nil {
		log.Printf("Error fetching overdue invoices: %v", err)
		return
	}

	if len(invoices) == 0 {
		log.Println("No overdue invoices for dunning")
		return
	}

	log.Printf("Processing %d overdue invoices for dunning", len(invoices))

	for _, invoice := range invoices {
		s.processInvoice(ctx, invoice)
	}
}

// processInvoice handles a single overdue invoice
func (s *DunningScheduler) processInvoice(ctx context.Context, invoice domain.OverdueInvoice) {
	daysOverdue := int(time.Since(invoice.DueDate).Hours() / 24)

	// Determine dunning level based on retry count
	level := invoice.RetryCount + 1
	if level > 3 {
		level = 3
	}

	// Check if we should mark as uncollectible
	if daysOverdue >= s.config.CancelAfterDays || invoice.RetryCount >= s.config.MaxRetries {
		if err := s.invoiceRepo.MarkAsUncollectible(ctx, invoice.ID); err != nil {
			log.Printf("Failed to mark invoice %s as uncollectible: %v", invoice.InvoiceNumber, err)
		} else {
			log.Printf("📛 Marked invoice %s as uncollectible (days overdue: %d)", invoice.InvoiceNumber, daysOverdue)
		}
		return
	}

	// Calculate next retry date
	nextRetryDays := s.getNextRetryDays(invoice.RetryCount)
	nextRetry := time.Now().AddDate(0, 0, nextRetryDays)
	suspensionDate := invoice.DueDate.AddDate(0, 0, s.config.SuspendAfterDays)

	// Send dunning email
	data := email.DunningEmailData{
		CustomerName:   invoice.CustomerName,
		CustomerEmail:  invoice.CustomerEmail,
		InvoiceNumber:  invoice.InvoiceNumber,
		Amount:         formatAmount(invoice.Amount, invoice.Currency),
		DaysOverdue:    daysOverdue,
		RetryCount:     invoice.RetryCount + 1,
		NextRetryDate:  nextRetry.Format("January 2, 2006"),
		SuspensionDate: suspensionDate.Format("January 2, 2006"),
		// Real SPA routes: hosted checkout pays the invoice; the portal login
		// (magic link) fronts the card-update flow. The previous /portal/pay and
		// /portal/payment-methods paths never existed.
		PayNowURL:        fmt.Sprintf("%s/checkout/%s", s.portalBaseURL, invoice.ID),
		UpdatePaymentURL: s.portalBaseURL + "/portal/login",
	}

	if err := s.notificationSvc.SendDunningEmail(ctx, level, data); err != nil {
		log.Printf("Failed to send dunning email for invoice %s: %v", invoice.InvoiceNumber, err)
	} else {
		log.Printf("📧 Sent dunning level %d email for invoice %s to %s", level, invoice.InvoiceNumber, invoice.CustomerEmail)
	}

	// Update retry info and hand off to RetryWorker for smart dunning
	if err := s.invoiceRepo.UpdateRetryInfoWithDunning(ctx, invoice.ID, nextRetry, invoice.RetryCount+1, "worker"); err != nil {
		log.Printf("Failed to update retry info for invoice %s: %v", invoice.InvoiceNumber, err)
	} else {
		log.Printf("Handed invoice %s to RetryWorker for smart dunning", invoice.InvoiceNumber)
	}
}

// getNextRetryDays returns the number of days until next retry
func (s *DunningScheduler) getNextRetryDays(currentRetryCount int) int {
	if currentRetryCount < len(s.config.RetryDays) {
		return s.config.RetryDays[currentRetryCount]
	}
	// Default to last configured interval
	return s.config.RetryDays[len(s.config.RetryDays)-1]
}
