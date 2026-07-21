package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// EUEInvoiceRetryWorker redrives delivery of EU e-invoice documents that failed
// to transmit to the Access Point. The document is generated once and is
// immutable, so this only re-transmits the stored document — it never
// regenerates. Generation failures (bad data) are not scheduled for retry and so
// are never picked up here; they surface for manual correction.
//
// Mirrors the India IRN retry worker: an atomic lease claim (safe across
// instances without Redis), an exponential backoff, and a permanent-fail cap.
type EUEInvoiceRetryWorker struct {
	store   euRetryStore
	service euRetransmitter
	logger  *slog.Logger
}

// euRetryStore is the narrow persistence the worker needs; satisfied by
// *db.EUInvoiceRepository.
type euRetryStore interface {
	ClaimFailedEUInvoices(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*domain.EUInvoice, error)
	UpdateDelivery(ctx context.Context, e *domain.EUInvoice) error
}

// euRetransmitter re-hands a stored document to the transport; satisfied by
// *service.EUEInvoiceService.
type euRetransmitter interface {
	RetryTransmission(ctx context.Context, rec *domain.EUInvoice) (*domain.EUInvoiceTransmission, error)
}

func NewEUEInvoiceRetryWorker(store euRetryStore, svc euRetransmitter) *EUEInvoiceRetryWorker {
	return &EUEInvoiceRetryWorker{
		store:   store,
		service: svc,
		logger:  slog.Default().With("worker", "eu_einvoice_retry"),
	}
}

// euEInvoiceClaimLease is how long a claimed row is hidden before it re-surfaces
// — long enough to outlast one Access Point round-trip so a crashed runner's row
// is retried rather than lost, but shorter than the smallest backoff so a genuine
// failure isn't delayed by the lease.
const euEInvoiceClaimLease = 2 * time.Minute

// euEInvoiceClaimBatch bounds how many rows one tick processes.
const euEInvoiceClaimBatch = 20

// Start runs the poll loop until ctx is cancelled.
func (w *EUEInvoiceRetryWorker) Start(ctx context.Context) {
	if w == nil || w.store == nil || w.service == nil {
		slog.Info("eu e-invoice retry worker not configured; skipping")
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	slog.Info("eu e-invoice retry worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("eu e-invoice retry worker stopping")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *EUEInvoiceRetryWorker) processRetries(ctx context.Context) {
	// Claim (not just read) due rows: the scheduler lock is a no-op without
	// Redis, so the atomic lease is what stops two instances from re-transmitting
	// the same document to the Access Point. UTC to match the TIMESTAMPTZ column.
	now := time.Now().UTC()
	recs, err := w.store.ClaimFailedEUInvoices(ctx, now, now.Add(euEInvoiceClaimLease), euEInvoiceClaimBatch)
	if err != nil {
		w.logger.Error("claim failed EU e-invoices", "error", err)
		return
	}
	if len(recs) == 0 {
		return
	}
	w.logger.Info("found EU e-invoices to redrive", "count", len(recs))

	for _, rec := range recs {
		w.retryOne(ctx, rec)
	}
}

// retryOne re-transmits a single claimed record. The worker holds the lease and
// is the sole writer, so it mutates the claimed row and writes it back directly —
// no read-modify-write race.
func (w *EUEInvoiceRetryWorker) retryOne(ctx context.Context, rec *domain.EUInvoice) {
	// Give up after the cap: clear the schedule so it stops cycling and leave it
	// failed for manual attention.
	if rec.RetryCount >= service.MaxEUEInvoiceRetries {
		rec.NextRetryAt = nil
		rec.ErrorMessage = "max delivery retries exceeded"
		w.logger.Warn("EU e-invoice permanently failed after max retries", "invoice_id", rec.InvoiceID, "retries", rec.RetryCount)
		if err := w.store.UpdateDelivery(ctx, rec); err != nil {
			w.logger.Error("persist permanent-fail", "invoice_id", rec.InvoiceID, "error", err)
		}
		return
	}

	res, err := w.service.RetryTransmission(ctx, rec)
	if err != nil || res == nil {
		// Failed again: advance the count and schedule the next attempt on the
		// backoff curve (last interval repeats once the curve is exhausted).
		rec.RetryCount++
		idx := rec.RetryCount - 1
		if idx >= len(service.EUEInvoiceBackoff) {
			idx = len(service.EUEInvoiceBackoff) - 1
		}
		next := time.Now().UTC().Add(service.EUEInvoiceBackoff[idx])
		rec.NextRetryAt = &next
		rec.Status = domain.EUInvoiceStatusFailed
		rec.ErrorMessage = "delivery retry failed"
		w.logger.Warn("EU e-invoice redrive failed, rescheduled", "invoice_id", rec.InvoiceID, "attempt", rec.RetryCount, "next_retry_at", next, "error", err)
		if uerr := w.store.UpdateDelivery(ctx, rec); uerr != nil {
			w.logger.Error("persist reschedule", "invoice_id", rec.InvoiceID, "error", uerr)
		}
		return
	}

	// Delivered: clear the schedule and record the transport message id.
	rec.Status = res.Status
	if rec.Status == "" {
		rec.Status = domain.EUInvoiceStatusSent
	}
	rec.MessageID = res.MessageID
	rec.ErrorMessage = ""
	rec.NextRetryAt = nil
	w.logger.Info("EU e-invoice delivered on retry", "invoice_id", rec.InvoiceID, "message_id", rec.MessageID)
	if err := w.store.UpdateDelivery(ctx, rec); err != nil {
		w.logger.Error("persist delivered", "invoice_id", rec.InvoiceID, "error", err)
	}
}
