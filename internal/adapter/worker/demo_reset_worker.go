package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// DemoResetWorker restores the public sandbox to its pristine seeded state
// on an interval (docs/spec_demo_mode.md D3): whatever visitors did in the
// last window disappears, so vandalism and clutter never persist. Runs
// only in DEMO_MODE.

// demoResetter is the service slice; *service.DemoService.
type demoResetter interface {
	Reset(ctx context.Context) error
}

type DemoResetWorker struct {
	resetter demoResetter
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

func NewDemoResetWorker(resetter demoResetter, interval time.Duration) *DemoResetWorker {
	return &DemoResetWorker{
		resetter: resetter,
		interval: interval,
		done:     make(chan bool),
	}
}

func (w *DemoResetWorker) Start() {
	w.ticker = time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-w.ticker.C:
				w.RunOnce(context.Background())
			}
		}
	}()
	slog.Info("demo reset worker started", "interval", w.interval)
}

func (w *DemoResetWorker) Stop() {
	w.stopOnce.Do(func() {
		if w.ticker != nil {
			w.ticker.Stop()
		}
		close(w.done)
		slog.Info("demo reset worker stopped")
	})
}

// RunOnce performs one reset; failures log and the next tick retries.
func (w *DemoResetWorker) RunOnce(ctx context.Context) {
	start := time.Now()
	if err := w.resetter.Reset(ctx); err != nil {
		slog.Error("demo reset failed (next tick retries)", "error", err)
		return
	}
	slog.Info("demo environment reset to pristine data", "took", time.Since(start).Round(time.Millisecond))
}
