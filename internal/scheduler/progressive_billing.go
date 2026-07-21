package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// ProgressiveBillingScheduler auto-triggers interim (progressive) billing (A5).
// Progressive subscriptions bill their metered usage incrementally: once accrued
// usage crosses the threshold, an interim invoice bills the delta. Without this
// sweep that only happens when someone calls POST /subscriptions/:id/bill-usage;
// the sweep makes it automatic.
//
// Each tick it lists active progressive subscriptions and asks the biller to
// generate an interim invoice for each. All correctness lives downstream:
// GenerateProgressiveInvoiceForSub is threshold-gated (bills nothing until the
// threshold is crossed) and every delta is claimed via the watermark
// compare-and-swap, so running the sweep on several instances — or overlapping a
// manual bill-usage call — can never double-bill. The distributed lock here is
// only an efficiency guard against redundant scans, not the safety mechanism.
type ProgressiveBillingScheduler struct {
	repo     progressiveSweepRepo
	biller   progressiveBiller
	locker   port.Locker
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

// progressiveSweepRepo lists the sweep's candidate subscriptions.
type progressiveSweepRepo interface {
	ListActiveProgressiveSubscriptionIDs(ctx context.Context) ([]uuid.UUID, error)
}

// progressiveBiller generates an interim invoice for one subscription when its
// accrued usage has crossed the threshold; it returns (nil, nil) when nothing
// is due. Satisfied by *service.InvoiceService.
type progressiveBiller interface {
	GenerateProgressiveInvoiceForSub(ctx context.Context, subID uuid.UUID) (*domain.Invoice, error)
}

// DefaultProgressiveSweepInterval is how often the sweep runs when the caller
// doesn't specify one. Hourly keeps interim invoices timely without hammering
// the DB; the threshold gate means most ticks bill nothing.
const DefaultProgressiveSweepInterval = time.Hour

// NewProgressiveBillingScheduler builds the sweep. interval <= 0 falls back to
// DefaultProgressiveSweepInterval.
func NewProgressiveBillingScheduler(repo progressiveSweepRepo, biller progressiveBiller, locker port.Locker, interval time.Duration) *ProgressiveBillingScheduler {
	if interval <= 0 {
		interval = DefaultProgressiveSweepInterval
	}
	return &ProgressiveBillingScheduler{
		repo:     repo,
		biller:   biller,
		locker:   locker,
		interval: interval,
		done:     make(chan bool),
	}
}

// Start begins the sweep loop (runs once immediately, then every interval).
func (s *ProgressiveBillingScheduler) Start() {
	s.ticker = time.NewTicker(s.interval)
	go s.run()
	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.run()
			}
		}
	}()
	slog.Info("progressive-billing sweep started", "interval", s.interval)
}

// Stop halts the sweep loop.
func (s *ProgressiveBillingScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("progressive-billing sweep stopped")
	})
}

// run sweeps every active progressive subscription once, billing the interim
// delta for any that have crossed the threshold.
func (s *ProgressiveBillingScheduler) run() {
	ctx := context.Background()

	// Efficiency guard only — the watermark CAS is what actually prevents
	// double-billing, so a missed lock is harmless (at worst a redundant scan).
	release, acquired, err := s.locker.Obtain(ctx, "scheduler:progressive-billing", 10*time.Minute)
	if err != nil {
		slog.Error("progressive sweep: failed to obtain lock", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("progressive sweep: failed to release lock", "error", err)
		}
	}()

	ids, err := s.repo.ListActiveProgressiveSubscriptionIDs(ctx)
	if err != nil {
		slog.Error("progressive sweep: failed to list subscriptions", "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	var billed int
	for _, id := range ids {
		inv, err := s.biller.GenerateProgressiveInvoiceForSub(ctx, id)
		if err != nil {
			slog.Error("progressive sweep: interim billing failed", "subscription_id", id, "error", err)
			continue
		}
		if inv != nil {
			billed++
			slog.Info("progressive sweep: interim invoice generated", "subscription_id", id, "invoice_id", inv.ID, "amount_due", inv.AmountDue)
		}
	}
	slog.Info("progressive sweep complete", "candidates", len(ids), "billed", billed)
}
