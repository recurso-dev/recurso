package worker

import (
	"context"
	"log"
	"time"

	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

type RetryWorker struct {
	invoiceRepo port.InvoiceRepository
	retryService *service.SmartRetryService
	gateway     port.PaymentGateway
	notifier    port.Notifier
}

func NewRetryWorker(
	invoiceRepo port.InvoiceRepository,
	retryService *service.SmartRetryService,
	gateway port.PaymentGateway,
	notifier port.Notifier,
) *RetryWorker {
	return &RetryWorker{
		invoiceRepo: invoiceRepo,
		retryService: retryService,
		gateway:     gateway,
		notifier:    notifier,
	}
}

// Start runs the worker loop.
// In production, this would be more robust (graceful shutdown, leader election usually).
func (w *RetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Poll every 10s for demo
	defer ticker.Stop()

	log.Println("RetryWorker started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("RetryWorker stopping...")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *RetryWorker) processRetries(ctx context.Context) {
	// 1. Fetch Invoices due for retry
	invoices, err := w.invoiceRepo.GetDueForRetry(ctx)
	if err != nil {
		log.Printf("Worker: Failed to fetch retry invoices: %v", err)
		return
	}
	
	if len(invoices) > 0 {
		log.Printf("Worker: Found %d invoices to retry", len(invoices))
	}

	for _, inv := range invoices {
		log.Printf("Worker: Retrying Invoice %s (Attempt %d)", inv.InvoiceNumber, inv.RetryCount+1)
		
		// 2. Attempt Payment (Mock Gateway Call)
		// Assuming we can use saved card or similar. Here we just try creation again or capture.
		// For P2, we simulate failure or success.
		// Let's assume 50% chance of success for demo purpose if this was real.
		// But let's verify if user wants to see success.
		
		// For now, let's just Log and Schedule Next Retry (simulate failure)
		// unless we mock a success trigger.
		
		// Increment count
		inv.RetryCount++
		
		// Calculate Next Time using AI Service
		nextTime := w.retryService.GetNextRetryTime(inv)
		
		if nextTime.IsZero() {
			log.Printf("Worker: Max retries reached for %s. Marking uncollectible.", inv.InvoiceNumber)
			inv.Status = domain.InvoiceStatusUncollectible
			inv.NextRetryAt = nil
		} else {
			log.Printf("Worker: Payment failed. Next retry calculated for: %v", nextTime)
			inv.NextRetryAt = &nextTime
			inv.Status = domain.InvoiceStatusPastDue
		}
		
		if err := w.invoiceRepo.Update(ctx, inv); err != nil {
			log.Printf("Worker: Failed to update invoice %s: %v", inv.ID, err)
		}
	}
}
