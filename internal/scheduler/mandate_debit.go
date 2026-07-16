package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// debitClaimWindow is how far ClaimDueForDebit pushes next_debit_at when it
// claims a due mandate. It is the failure-retry lease: shorter than the 1h tick
// so a failed debit retries on the next tick, but far longer than a single
// gateway debit takes so a mandate being processed is never re-claimed.
const debitClaimWindow = 15 * time.Minute

type MandateDebitScheduler struct {
	mandateRepo port.MandateRepository
	mandateSvc  *service.MandateService
	locker      port.Locker
	ticker      *time.Ticker
	done        chan bool
	stopOnce    sync.Once
}

func NewMandateDebitScheduler(
	mandateRepo port.MandateRepository,
	mandateSvc *service.MandateService,
	locker port.Locker,
) *MandateDebitScheduler {
	return &MandateDebitScheduler{
		mandateRepo: mandateRepo,
		mandateSvc:  mandateSvc,
		locker:      locker,
		done:        make(chan bool),
	}
}

func (s *MandateDebitScheduler) Start() {
	s.ticker = time.NewTicker(1 * time.Hour)

	go s.runDebits()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runDebits()
			}
		}
	}()

	slog.Info("mandate debit scheduler started (runs hourly)")
}

func (s *MandateDebitScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("mandate debit scheduler stopped")
	})
}

func (s *MandateDebitScheduler) runDebits() {
	ctx := context.Background()

	lockKey := "scheduler:mandate-debit"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for mandate debit scheduler", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for mandate debit scheduler", "error", err)
		}
	}()

	// Atomically CLAIM due mandates (not just read them): the distributed lock
	// above is a no-op without Redis, so the claim is what actually guarantees a
	// mandate is charged by exactly one runner (ENG-161).
	mandates, err := s.mandateRepo.ClaimDueForDebit(ctx, debitClaimWindow)
	if err != nil {
		slog.Error("failed to claim mandates ready for debit", "error", err)
		return
	}

	if len(mandates) == 0 {
		return
	}

	slog.Info("mandates ready for debit", "count", len(mandates))

	for _, mandate := range mandates {
		// Charge the subscription's real recurring amount (plan price + tax),
		// NOT mandate.MaxAmount — MaxAmount is the authorization ceiling and
		// debiting it over-charged ~2× every cycle (ENG-165).
		if err := s.mandateSvc.DebitSubscription(ctx, mandate); err != nil {
			slog.Error("failed to execute debit for mandate", "mandate_id", mandate.ID, "error", err)
			continue
		}
		slog.Info("successfully debited mandate", "mandate_id", mandate.ID)
	}
}
