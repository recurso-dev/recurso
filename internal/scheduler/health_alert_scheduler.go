package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/adapter/alerting"
)

// ComponentCheck probes one dependency the /health endpoint reports on
// (postgres, redis, tigerbeetle). Check returns nil when the component is
// healthy. Severity is the severity of the "degraded" alert fired when the
// component goes down (recoveries always fire as info).
type ComponentCheck struct {
	Name     string
	Severity alerting.Severity
	Check    func(ctx context.Context) error
}

// HealthAlertScheduler is the solo-operator safety net: every interval it
// evaluates the same component checks the /health endpoint uses and sends an
// alert on state TRANSITIONS only — up→down fires "<name> degraded" once,
// down→up fires "<name> recovered" once. Steady state never re-fires, so a
// component that stays down all night produces exactly one alert.
//
// Components start assumed-up, so a dependency that is already down at the
// first evaluation fires one degraded alert (~interval after boot).
//
// Unlike the job schedulers this runs on every replica without a distributed
// lock, on purpose: connectivity is a per-instance property, and each
// instance should report its own view.
type HealthAlertScheduler struct {
	alerter  alerting.Alerter
	checks   []ComponentCheck
	interval time.Duration
	lastDown map[string]bool // component name → was down at the previous evaluation
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

// NewHealthAlertScheduler creates the watcher. interval <= 0 defaults to 60s.
func NewHealthAlertScheduler(alerter alerting.Alerter, checks []ComponentCheck, interval time.Duration) *HealthAlertScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &HealthAlertScheduler{
		alerter:  alerter,
		checks:   checks,
		interval: interval,
		lastDown: make(map[string]bool),
		done:     make(chan bool),
	}
}

// Start begins evaluating the component checks every interval. The first
// evaluation happens one full interval after start (not immediately), so a
// stack still booting doesn't false-alarm.
func (s *HealthAlertScheduler) Start() {
	s.ticker = time.NewTicker(s.interval)

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.evaluate(context.Background())
			}
		}
	}()

	slog.Info("Health alert watcher started",
		"interval", s.interval, "components", len(s.checks))
}

// Stop stops the watcher. Safe to call more than once.
func (s *HealthAlertScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("Health alert watcher stopped")
	})
}

// evaluate probes every component once and alerts on transitions only.
// It is called from a single goroutine, so lastDown needs no locking.
func (s *HealthAlertScheduler) evaluate(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, c := range s.checks {
		err := c.Check(ctx)
		down := err != nil
		wasDown := s.lastDown[c.Name]
		s.lastDown[c.Name] = down

		switch {
		case down && !wasDown:
			slog.Warn("Health watcher: component degraded",
				"component", c.Name, "error", err)
			s.send(ctx, c.Severity,
				fmt.Sprintf("%s degraded", c.Name),
				fmt.Sprintf("health check failed: %v", err))
		case !down && wasDown:
			slog.Info("Health watcher: component recovered", "component", c.Name)
			s.send(ctx, alerting.SeverityInfo,
				fmt.Sprintf("%s recovered", c.Name),
				"health check passing again")
		}
	}
}

// send fires one alert and logs delivery failures. It never retries:
// the next transition (not the next tick) is the next attempt.
func (s *HealthAlertScheduler) send(ctx context.Context, severity alerting.Severity, title, body string) {
	if err := s.alerter.Send(ctx, severity, title, body); err != nil {
		slog.Error("Health watcher: failed to deliver alert",
			"title", title, "error", err)
	}
}
