package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// BillingCycleScheduler drives unattended subscription renewal
// (Lago-parity A1): every tick it asks the renewal engine to claim and
// process due, locally-billed subscriptions. Mandate- and gateway-managed
// subscriptions are excluded at the claim query, so this composes with the
// mandate-debit scheduler rather than racing it.
//
// The distributed lock is an optimization (skip redundant sweeps); the
// repository's leased claim is the actual at-most-once guarantee, exactly
// as with mandate debits (ENG-161).

// renewalProcessor is the slice of service.RenewalService the scheduler
// needs.
type renewalProcessor interface {
	ProcessDueRenewals(ctx context.Context) (int, error)
}

type BillingCycleScheduler struct {
	renewals renewalProcessor
	locker   port.Locker
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

func NewBillingCycleScheduler(renewals renewalProcessor, locker port.Locker, interval time.Duration) *BillingCycleScheduler {
	return &BillingCycleScheduler{
		renewals: renewals,
		locker:   locker,
		interval: interval,
		done:     make(chan bool),
	}
}

func (s *BillingCycleScheduler) Start() {
	s.ticker = time.NewTicker(s.interval)

	go s.runRenewals()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runRenewals()
			}
		}
	}()

	slog.Info("billing cycle scheduler started", "interval", s.interval)
}

func (s *BillingCycleScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("billing cycle scheduler stopped")
	})
}

func (s *BillingCycleScheduler) runRenewals() {
	ctx := context.Background()

	release, acquired, err := s.locker.Obtain(ctx, "scheduler:billing-cycle", 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for billing cycle scheduler", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for billing cycle scheduler", "error", err)
		}
	}()

	renewed, err := s.renewals.ProcessDueRenewals(ctx)
	if err != nil {
		slog.Error("billing cycle sweep failed", "error", err)
		return
	}
	if renewed > 0 {
		slog.Info("billing cycle sweep complete", "renewed", renewed)
	}
}
