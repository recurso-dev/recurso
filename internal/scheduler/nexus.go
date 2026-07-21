package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// NexusScheduler evaluates US economic-nexus thresholds daily for every
// tenant, so a crossing establishes nexus even when nobody opens the status
// page (ENG-16 Phase 2). The status service takes tenant ids as explicit
// parameters, so this scheduler is immune to the tenant-context bug class.
type NexusScheduler struct {
	tenantRepo *db.TenantRepository
	status     *service.NexusStatusService
	alerts     *service.NexusAlertService // optional (Track D · D1): proactive threshold alerts
	locker     port.Locker
	ticker     *time.Ticker
	done       chan bool
	stopOnce   sync.Once
}

// SetAlertService enables proactive economic-nexus threshold alerting: when set,
// the daily run notifies each tenant as they near or cross a state threshold
// (and still establishes economic nexus, since alerting computes full status).
// Nil-safe optional-dependency wiring — without it the scheduler only establishes.
func (s *NexusScheduler) SetAlertService(a *service.NexusAlertService) {
	s.alerts = a
}

func NewNexusScheduler(tenantRepo *db.TenantRepository, status *service.NexusStatusService, locker port.Locker) *NexusScheduler {
	return &NexusScheduler{
		tenantRepo: tenantRepo,
		status:     status,
		locker:     locker,
		done:       make(chan bool),
	}
}

// Start begins the daily evaluation (also runs once at boot).
func (s *NexusScheduler) Start() {
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

	slog.Info("nexus scheduler started (runs daily)")
}

func (s *NexusScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("nexus scheduler stopped")
	})
}

func (s *NexusScheduler) run() {
	ctx := context.Background()

	release, acquired, err := s.locker.Obtain(ctx, "scheduler:nexus", 10*time.Minute)
	if err != nil {
		slog.Error("failed to obtain lock for nexus scheduler", "error", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("failed to release lock for nexus scheduler", "error", err)
		}
	}()

	tenants, err := s.tenantRepo.ListTenants(ctx)
	if err != nil {
		slog.Error("nexus scheduler: failed to list tenants", "error", err)
		return
	}

	year := time.Now().UTC().Year()
	for _, t := range tenants {
		// When alerting is wired, EvaluateAndAlert computes full per-state status
		// (which establishes new economic nexus) and notifies on newly reached
		// approaching/crossed thresholds. Otherwise, just establish.
		if s.alerts != nil {
			if err := s.alerts.EvaluateAndAlert(ctx, t.ID, year); err != nil {
				slog.Error("nexus scheduler: evaluate+alert failed for tenant", "tenant_id", t.ID, "error", err)
			}
			continue
		}

		established, err := s.status.EvaluateEconomicNexus(ctx, t.ID, year)
		if err != nil {
			slog.Error("nexus scheduler: evaluation failed for tenant", "tenant_id", t.ID, "error", err)
			continue
		}
		for _, state := range established {
			slog.Info("economic nexus established", "tenant_id", t.ID, "state", state)
		}
	}
}
