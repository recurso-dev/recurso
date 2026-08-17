package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// CloudUsageMeasurer measures the current period's Recurso Cloud usage across
// all tenants. Satisfied by *service.CloudUsageService.
type CloudUsageMeasurer interface {
	MeasureCurrentPeriod(ctx context.Context) (int, error)
}

// CloudUsageScheduler re-measures every tenant's Recurso Cloud usage (tracked
// revenue + collected volume) once a day. Measurement is an idempotent upsert
// per (tenant, period, currency), so a missed or repeated run only refreshes
// the current month's rows. It never moves money.
type CloudUsageScheduler struct {
	measurer CloudUsageMeasurer
	locker   port.Locker
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

func NewCloudUsageScheduler(measurer CloudUsageMeasurer, locker port.Locker) *CloudUsageScheduler {
	return &CloudUsageScheduler{
		measurer: measurer,
		locker:   locker,
		done:     make(chan bool),
	}
}

// Start measures immediately and then every 24 hours.
func (s *CloudUsageScheduler) Start() {
	s.ticker = time.NewTicker(24 * time.Hour)
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
	slog.Info("Recurso Cloud usage scheduler started (runs every 24 hours)")
}

// Stop stops the scheduler. Safe to call more than once.
func (s *CloudUsageScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("Recurso Cloud usage scheduler stopped")
	})
}

func (s *CloudUsageScheduler) run() {
	ctx := context.Background()

	release, acquired, err := s.locker.Obtain(ctx, "scheduler:cloud-usage", 30*time.Minute)
	if err != nil {
		slog.Error("cloud usage: failed to obtain lock", "error", err)
		return
	}
	if !acquired {
		return // another instance holds it
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("cloud usage: failed to release lock", "error", err)
		}
	}()

	n, err := s.measurer.MeasureCurrentPeriod(ctx)
	if err != nil {
		slog.Error("cloud usage measurement failed", "error", err)
		return
	}
	slog.Info("Recurso Cloud usage measured", "readings", n)
}
