package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

type MandateDebitScheduler struct {
	mandateRepo port.MandateRepository
	mandateSvc  *service.MandateService
	locker      port.Locker
	ticker      *time.Ticker
	done        chan bool
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

	log.Println("Mandate debit scheduler started (runs hourly)")
}

func (s *MandateDebitScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("Mandate debit scheduler stopped")
}

func (s *MandateDebitScheduler) runDebits() {
	ctx := context.Background()

	lockKey := "scheduler:mandate-debit"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 10*time.Minute)
	if err != nil {
		log.Printf("Failed to obtain lock for mandate debit scheduler: %v", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			log.Printf("Failed to release lock for mandate debit scheduler: %v", err)
		}
	}()

	mandates, err := s.mandateRepo.GetReadyForDebit(ctx)
	if err != nil {
		log.Printf("Error fetching mandates ready for debit: %v", err)
		return
	}

	if len(mandates) == 0 {
		return
	}

	log.Printf("Found %d mandates ready for debit", len(mandates))

	for _, mandate := range mandates {
		if err := s.mandateSvc.ExecuteDebit(ctx, mandate, mandate.MaxAmount, "INR"); err != nil {
			log.Printf("Failed to execute debit for mandate %s: %v", mandate.ID, err)
			continue
		}
		log.Printf("Successfully debited mandate %s", mandate.ID)
	}
}
