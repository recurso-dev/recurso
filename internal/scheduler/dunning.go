package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
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
	stopOnce        sync.Once
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

	slog.Info("dunning scheduler started (runs every 6 hours)")
}

// Stop stops the scheduler
func (s *DunningScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("dunning scheduler stopped")
	})
}

// processDunning handles all overdue invoices
func (s *DunningScheduler) processDunning() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:dunning"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 30*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for dunning scheduler", "error", err)
		return
	}
	if !acquired {
		return // Lock held by another instance
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for dunning scheduler", "error", err)
		}
	}()

	invoices, err := s.invoiceRepo.GetOverdueInvoices(ctx)
	if err != nil {
		slog.Error("failed to fetch overdue invoices", "error", err)
		return
	}

	if len(invoices) == 0 {
		slog.Info("no overdue invoices for dunning")
		return
	}

	slog.Info("processing overdue invoices for dunning", "count", len(invoices))

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
			slog.Error("failed to mark invoice as uncollectible", "invoice_number", invoice.InvoiceNumber, "error", err)
		} else {
			slog.Info("marked invoice as uncollectible", "invoice_number", invoice.InvoiceNumber, "days_overdue", daysOverdue)
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
		slog.Error("failed to send dunning email", "invoice_number", invoice.InvoiceNumber, "error", err)
	} else {
		slog.Info("sent dunning email", "level", level, "invoice_number", invoice.InvoiceNumber, "customer_email", invoice.CustomerEmail)
	}

	// Hand off to the RetryWorker for smart dunning — EXCEPT UPI-mandate invoices.
	// The worker's gateway retry (RetryPayment) can't collect a mandate: it just
	// creates a dangling order and never debits the mandate token (ENG-168). And
	// auto-retrying via the mandate itself is unsafe — Razorpay ignores the
	// idempotency header and captures async, so a re-attempt double-charges. So
	// keep mandate invoices on the scheduler's email-dunning path (the pay-now
	// link lets the customer settle interactively; the mandate charges the next
	// cycle) and let them escalate to uncollectible here like any other invoice.
	managedBy := "worker"
	if invoice.IsMandate {
		managedBy = "scheduler"
	}
	if err := s.invoiceRepo.UpdateRetryInfoWithDunning(ctx, invoice.ID, nextRetry, invoice.RetryCount+1, managedBy); err != nil {
		slog.Error("failed to update retry info for invoice", "invoice_number", invoice.InvoiceNumber, "error", err)
	} else if managedBy == "worker" {
		slog.Info("handed invoice to retry worker for smart dunning", "invoice_number", invoice.InvoiceNumber)
	} else {
		slog.Info("mandate invoice kept on email dunning (no gateway auto-retry)", "invoice_number", invoice.InvoiceNumber)
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
