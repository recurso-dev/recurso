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

// walletMaintainer is the slice of service.WalletService the sweep runs:
// expiring dated promotional credit and topping up below-threshold wallets
// (Lago-parity B1). nil-safe.
type walletMaintainer interface {
	ExpireOverdueCredits(ctx context.Context) (int, error)
	ProcessAutoRecharges(ctx context.Context) (int, error)
}

// creditMaintainer is the slice of service.CreditNoteService the sweep runs:
// writing off dated account credits whose expiry has passed (ledger-backed
// credits inc 2). nil-safe.
type creditMaintainer interface {
	ExpireDueCredits(ctx context.Context) (int, error)
}

type BillingCycleScheduler struct {
	renewals renewalProcessor
	wallets  walletMaintainer // nil-safe
	credits  creditMaintainer // nil-safe
	alerts   alertEvaluator   // nil-safe
	locker   port.Locker
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

// SetWalletMaintainer wires wallet expiry + auto-recharge into the sweep.
func (s *BillingCycleScheduler) SetWalletMaintainer(w walletMaintainer) { s.wallets = w }

// SetCreditMaintainer wires account-credit expiry into the sweep.
func (s *BillingCycleScheduler) SetCreditMaintainer(c creditMaintainer) { s.credits = c }

// alertEvaluator is the slice of service.UsageAlertService the sweep runs
// (Lago-parity B3). nil-safe.
type alertEvaluator interface {
	EvaluateAlerts(ctx context.Context) (int, error)
}

// SetAlertEvaluator wires usage-threshold evaluation into the sweep.
func (s *BillingCycleScheduler) SetAlertEvaluator(a alertEvaluator) { s.alerts = a }

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

	// Wallet maintenance rides the same tick: write off expired promotional
	// credit BEFORE evaluating recharges, so a wallet whose balance was
	// mostly expired credit tops up in the same sweep.
	if s.wallets != nil {
		if expired, err := s.wallets.ExpireOverdueCredits(ctx); err != nil {
			slog.Error("wallet expiry sweep failed", "error", err)
		} else if expired > 0 {
			slog.Info("expired wallet credit written off", "wallets", expired)
		}
		if recharged, err := s.wallets.ProcessAutoRecharges(ctx); err != nil {
			slog.Error("wallet auto-recharge sweep failed", "error", err)
		} else if recharged > 0 {
			slog.Info("wallets auto-recharged", "count", recharged)
		}
	}

	// Account-credit expiry rides the same tick: write off dated adjustment
	// credits whose expiry has passed (ledger-backed credits inc 2).
	if s.credits != nil {
		if expired, err := s.credits.ExpireDueCredits(ctx); err != nil {
			slog.Error("credit expiry sweep failed", "error", err)
		} else if expired > 0 {
			slog.Info("expired account credit written off", "credits", expired)
		}
	}

	// Usage threshold alerts (B3): evaluated on the same tick; the
	// per-period MarkFired claim keeps concurrent sweeps to one firing.
	if s.alerts != nil {
		if fired, err := s.alerts.EvaluateAlerts(ctx); err != nil {
			slog.Error("usage alert sweep failed", "error", err)
		} else if fired > 0 {
			slog.Info("usage alerts fired", "count", fired)
		}
	}
}
