package scheduler

import (
	"context"
	"log"
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
	locker     port.Locker
	ticker     *time.Ticker
	done       chan bool
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

	log.Println("✅ Nexus scheduler started (runs daily)")
}

func (s *NexusScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("🛑 Nexus scheduler stopped")
}

func (s *NexusScheduler) run() {
	ctx := context.Background()

	release, acquired, err := s.locker.Obtain(ctx, "scheduler:nexus", 10*time.Minute)
	if err != nil {
		log.Printf("Failed to obtain lock for nexus scheduler: %v", err)
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := release(ctx); err != nil {
			log.Printf("Failed to release lock for nexus scheduler: %v", err)
		}
	}()

	tenants, err := s.tenantRepo.ListTenants(ctx)
	if err != nil {
		log.Printf("Nexus scheduler: failed to list tenants: %v", err)
		return
	}

	year := time.Now().UTC().Year()
	for _, t := range tenants {
		established, err := s.status.EvaluateEconomicNexus(ctx, t.ID, year)
		if err != nil {
			log.Printf("Nexus scheduler: evaluation failed for tenant %s: %v", t.ID, err)
			continue
		}
		for _, state := range established {
			log.Printf("📍 Economic nexus established for tenant %s in %s", t.ID, state)
		}
	}
}
