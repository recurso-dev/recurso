package worker

import (
	"context"
	"log/slog"
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

// einvoiceClaimLease is how long ClaimFailedEInvoices holds a claimed row before
// it re-surfaces — longer than one GSP round-trip so a crashed runner's invoice
// is retried rather than lost, but shorter than the retry backoff.
const einvoiceClaimLease = 15 * time.Minute

// Start runs the worker loop, polling every 30 seconds.
func (w *EInvoiceRetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	slog.Info("e-invoice retry worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("e-invoice retry worker stopping")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *EInvoiceRetryWorker) processRetries(ctx context.Context) {
	// CLAIM (not just read) the due failed e-invoices: the scheduler lock is a
	// no-op without Redis, so the atomic lease is what stops two instances from
	// both re-submitting the same invoice to the government IRN endpoint.
	now := time.Now().UTC()
	invoices, err := w.invoiceRepo.ClaimFailedEInvoices(ctx, now, now.Add(einvoiceClaimLease), 20)
	if err != nil {
		slog.Error("e-invoice retry worker: failed to fetch failed e-invoices", "error", err)
		return
	}

	if len(invoices) == 0 {
		return
	}

	slog.Info("e-invoice retry worker: found e-invoices to retry", "count", len(invoices))

	for _, inv := range invoices {
		slog.Info("e-invoice retry worker: retrying e-invoice", "invoice_number", inv.InvoiceNumber, "attempt", inv.EInvoiceRetryCount+1)

		// Check max retries
		if inv.EInvoiceRetryCount >= maxEInvoiceRetries {
			slog.Warn("e-invoice retry worker: max retries reached, marking permanently failed", "invoice_number", inv.InvoiceNumber)
			inv.EInvoiceNextRetryAt = nil
			inv.EInvoiceErrorMessage = "max retries exceeded"
			if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
				slog.Error("e-invoice retry worker: failed to update invoice", "invoice_id", inv.ID, "error", updateErr)
			}
			continue
		}

		// Attempt retry via EInvoiceService. The service now fetches the invoice
		// tenant-scoped, so inject this invoice's tenant into the context (the
		// global poller has none of its own).
		tctx := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
		_, retryErr := w.einvoiceService.RetryFailedEInvoice(tctx, inv.ID)
		if retryErr != nil {
			// RetryFailedEInvoice re-fetched the invoice and persisted an
			// INCREMENTED retry count. Re-read that fresh copy before scheduling
			// the next retry — mutating our stale `inv` and writing it back would
			// clobber the increment (the count would never advance, so
			// maxEInvoiceRetries never fires and backoff never escalates).
			fresh, ferr := w.invoiceRepo.GetByIDPublic(ctx, inv.ID)
			if ferr != nil || fresh == nil {
				slog.Error("e-invoice retry worker: could not re-read invoice after retry", "invoice_id", inv.ID, "error", ferr)
				continue
			}

			backoffIdx := fresh.EInvoiceRetryCount
			if backoffIdx >= len(einvoiceBackoffDurations) {
				backoffIdx = len(einvoiceBackoffDurations) - 1
			}
			// e_invoice_next_retry_at is a timezone-naive TIMESTAMP and the claim
			// query reads it in UTC (now := time.Now().UTC()), so it MUST be
			// written in UTC too — a local-time write skews the due comparison by
			// the server's UTC offset (correct only by luck on a UTC host).
			nextRetry := time.Now().UTC().Add(einvoiceBackoffDurations[backoffIdx])
			fresh.EInvoiceNextRetryAt = &nextRetry

			slog.Warn("e-invoice retry worker: retry failed, rescheduled", "invoice_number", fresh.InvoiceNumber, "attempt", fresh.EInvoiceRetryCount, "next_retry_at", nextRetry)

			if updateErr := w.invoiceRepo.Update(ctx, fresh); updateErr != nil {
				slog.Error("e-invoice retry worker: failed to update invoice", "invoice_id", inv.ID, "error", updateErr)
			}
		} else {
			slog.Info("e-invoice retry worker: successfully generated e-invoice", "invoice_number", inv.InvoiceNumber)
		}
	}
}
