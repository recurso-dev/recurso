package worker

import (
	"context"
	"log"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// EInvoiceRetryWorker polls for FAILED e-invoices and retries them with exponential backoff.
type EInvoiceRetryWorker struct {
	invoiceRepo     port.InvoiceRepository
	einvoiceService *service.EInvoiceService
}

func NewEInvoiceRetryWorker(
	invoiceRepo port.InvoiceRepository,
	einvoiceService *service.EInvoiceService,
) *EInvoiceRetryWorker {
	return &EInvoiceRetryWorker{
		invoiceRepo:     invoiceRepo,
		einvoiceService: einvoiceService,
	}
}

// backoff intervals: 5min, 15min, 1hr, 6hr, 24hr
var einvoiceBackoffDurations = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	1 * time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

const maxEInvoiceRetries = 5

// Start runs the worker loop, polling every 30 seconds.
func (w *EInvoiceRetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("EInvoiceRetryWorker started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("EInvoiceRetryWorker stopping...")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *EInvoiceRetryWorker) processRetries(ctx context.Context) {
	invoices, err := w.invoiceRepo.GetFailedEInvoices(ctx)
	if err != nil {
		log.Printf("EInvoiceRetryWorker: Failed to fetch failed e-invoices: %v", err)
		return
	}

	if len(invoices) == 0 {
		return
	}

	log.Printf("EInvoiceRetryWorker: Found %d e-invoices to retry", len(invoices))

	for _, inv := range invoices {
		log.Printf("EInvoiceRetryWorker: Retrying e-invoice for %s (attempt %d)", inv.InvoiceNumber, inv.EInvoiceRetryCount+1)

		// Check max retries
		if inv.EInvoiceRetryCount >= maxEInvoiceRetries {
			log.Printf("EInvoiceRetryWorker: Max retries reached for %s. Marking as permanently FAILED.", inv.InvoiceNumber)
			inv.EInvoiceNextRetryAt = nil
			inv.EInvoiceErrorMessage = "max retries exceeded"
			if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
				log.Printf("EInvoiceRetryWorker: Failed to update invoice %s: %v", inv.ID, updateErr)
			}
			continue
		}

		// Attempt retry via EInvoiceService. The service now fetches the invoice
		// tenant-scoped, so inject this invoice's tenant into the context (the
		// global poller has none of its own).
		tctx := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
		_, retryErr := w.einvoiceService.RetryFailedEInvoice(tctx, inv.ID)
		if retryErr != nil {
			// Schedule next retry with exponential backoff
			backoffIdx := inv.EInvoiceRetryCount
			if backoffIdx >= len(einvoiceBackoffDurations) {
				backoffIdx = len(einvoiceBackoffDurations) - 1
			}
			nextRetry := time.Now().Add(einvoiceBackoffDurations[backoffIdx])
			inv.EInvoiceNextRetryAt = &nextRetry

			log.Printf("EInvoiceRetryWorker: Retry failed for %s. Next retry at %v", inv.InvoiceNumber, nextRetry)

			if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
				log.Printf("EInvoiceRetryWorker: Failed to update invoice %s: %v", inv.ID, updateErr)
			}
		} else {
			log.Printf("EInvoiceRetryWorker: Successfully generated e-invoice for %s", inv.InvoiceNumber)
		}
	}
}
