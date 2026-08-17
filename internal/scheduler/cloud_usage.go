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

// CloudChargePreviewer computes the money-free dry-run charge preview for the
// current period. Optional; satisfied by *service.CloudChargeService. Run right
// after measuring so the preview reflects the freshest readings.
type CloudChargePreviewer interface {
	PreviewCurrentPeriod(ctx context.Context) error
}

// CloudUsageScheduler re-measures every tenant's Recurso Cloud usage (tracked
// revenue + collected volume) once a day. Measurement is an idempotent upsert
// per (tenant, period, currency), so a missed or repeated run only refreshes
// the current month's rows. It never moves money.
type CloudUsageScheduler struct {
	measurer  CloudUsageMeasurer
	previewer CloudChargePreviewer // optional (Increment 3 dry-run); nil-safe
	locker    port.Locker
	ticker    *time.Ticker
	done      chan bool
	stopOnce  sync.Once
}

func NewCloudUsageScheduler(measurer CloudUsageMeasurer, locker port.Locker) *CloudUsageScheduler {
	return &CloudUsageScheduler{
		measurer: measurer,
		locker:   locker,
		done:     make(chan bool),
	}
}

// SetPreviewer wires the dry-run charge preview to run after each measurement.
// Money-free; leave unset to only measure.
func (s *CloudUsageScheduler) SetPreviewer(p CloudChargePreviewer) { s.previewer = p }

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

	// Money-free dry-run: recompute what each tenant WOULD be charged off the
	// fresh readings. Never bills — a failure here doesn't affect the meter.
	if s.previewer != nil {
		if err := s.previewer.PreviewCurrentPeriod(ctx); err != nil {
			slog.Error("cloud charge preview failed", "error", err)
		}
	}
}
