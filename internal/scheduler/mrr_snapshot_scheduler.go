package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// TenantRepoForMRRSnapshot is the narrow tenant listing the scheduler needs.
type TenantRepoForMRRSnapshot interface {
	ListTenants(ctx context.Context) ([]*domain.Tenant, error)
}

// MRRSnapshotCapturer records one tenant's MRR snapshot for a date.
type MRRSnapshotCapturer interface {
	CaptureMRRSnapshot(ctx context.Context, tenantID uuid.UUID, date time.Time) (int, error)
}

// MRRSnapshotScheduler captures every tenant's per-subscription MRR once a day,
// building the history the MRR waterfall diffs. Capture is idempotent per day,
// so a missed or repeated run only rewrites that day's rows.
type MRRSnapshotScheduler struct {
	tenantRepo TenantRepoForMRRSnapshot
	capturer   MRRSnapshotCapturer
	locker     port.Locker
	ticker     *time.Ticker
	done       chan bool
	stopOnce   sync.Once
}

func NewMRRSnapshotScheduler(tenantRepo TenantRepoForMRRSnapshot, capturer MRRSnapshotCapturer, locker port.Locker) *MRRSnapshotScheduler {
	return &MRRSnapshotScheduler{
		tenantRepo: tenantRepo,
		capturer:   capturer,
		locker:     locker,
		done:       make(chan bool),
	}
}

// Start runs the capture immediately and then every 24 hours.
func (s *MRRSnapshotScheduler) Start() {
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
	slog.Info("MRR snapshot scheduler started (runs every 24 hours)")
}

// Stop stops the scheduler. Safe to call more than once.
func (s *MRRSnapshotScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("MRR snapshot scheduler stopped")
	})
}

func (s *MRRSnapshotScheduler) run() {
	ctx := context.Background()

	release, acquired, err := s.locker.Obtain(ctx, "scheduler:mrr-snapshot", 30*time.Minute)
	if err != nil {
		slog.Error("MRR snapshot: failed to obtain lock", "error", err)
		return
	}
	if !acquired {
		return // another instance holds it
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("MRR snapshot: failed to release lock", "error", err)
		}
	}()

	tenants, err := s.tenantRepo.ListTenants(ctx)
	if err != nil {
		slog.Error("MRR snapshot: failed to list tenants", "error", err)
		return
	}
	now := time.Now()
	for _, tenant := range tenants {
		n, err := s.capturer.CaptureMRRSnapshot(ctx, tenant.ID, now)
		if err != nil {
			slog.Error("MRR snapshot capture failed for tenant", "tenant_id", tenant.ID, "error", err)
			continue
		}
		slog.Info("MRR snapshot captured", "tenant_id", tenant.ID, "subscriptions", n)
	}
}
