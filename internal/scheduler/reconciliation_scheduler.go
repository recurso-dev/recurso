package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

// TenantRepoForReconciliation is the narrow tenant listing the scheduler needs.
type TenantRepoForReconciliation interface {
	ListTenants(ctx context.Context) ([]*domain.Tenant, error)
}

// ReconciliationRunner runs a ledger reconciliation for one tenant.
type ReconciliationRunner interface {
	Run(ctx context.Context, tenantID uuid.UUID) (*service.ReconciliationReport, error)
}

// ReconciliationScheduler runs a daily ledger-vs-billing reconciliation for
// every tenant and warns when the books disagree.
type ReconciliationScheduler struct {
	tenantRepo TenantRepoForReconciliation
	runner     ReconciliationRunner
	locker     port.Locker
	ticker     *time.Ticker
	done       chan bool
	stopOnce   sync.Once
}

// NewReconciliationScheduler creates a new reconciliation scheduler.
func NewReconciliationScheduler(
	tenantRepo TenantRepoForReconciliation,
	runner ReconciliationRunner,
	locker port.Locker,
) *ReconciliationScheduler {
	return &ReconciliationScheduler{
		tenantRepo: tenantRepo,
		runner:     runner,
		locker:     locker,
		done:       make(chan bool),
	}
}

// Start begins the scheduler (runs every 24 hours, and once on start).
func (s *ReconciliationScheduler) Start() {
	s.ticker = time.NewTicker(24 * time.Hour)

	// Run immediately on start
	go s.runReconciliation()

	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.runReconciliation()
			}
		}
	}()

	slog.Info("Ledger reconciliation scheduler started (runs every 24 hours)")
}

// Stop stops the scheduler. Safe to call more than once.
func (s *ReconciliationScheduler) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.done)
		slog.Info("Ledger reconciliation scheduler stopped")
	})
}

// runReconciliation reconciles every tenant's ledger under a distributed lock.
func (s *ReconciliationScheduler) runReconciliation() {
	ctx := context.Background()

	// Distributed Lock: Prevent multiple instances from running this job
	lockKey := "scheduler:ledger-reconciliation"
	release, acquired, err := s.locker.Obtain(ctx, lockKey, 30*time.Minute)
	if err != nil {
		slog.Error("Failed to obtain lock for reconciliation scheduler", "error", err)
		return
	}
	if !acquired {
		return // Lock held by another instance
	}
	defer func() {
		if err := release(ctx); err != nil {
			slog.Error("Failed to release lock for reconciliation scheduler", "error", err)
		}
	}()

	tenants, err := s.tenantRepo.ListTenants(ctx)
	if err != nil {
		slog.Error("Reconciliation: failed to list tenants", "error", err)
		return
	}

	for _, tenant := range tenants {
		report, err := s.runner.Run(ctx, tenant.ID)
		if err != nil {
			slog.Error("Ledger reconciliation failed for tenant",
				"tenant_id", tenant.ID, "error", err)
			continue
		}

		if report.TotalDiscrepancies > 0 {
			slog.Warn("Ledger reconciliation found discrepancies",
				"tenant_id", tenant.ID,
				"total_discrepancies", report.TotalDiscrepancies,
				"invoices_checked", report.InvoicesChecked,
				"paid_invoices_checked", report.PaidInvoicesChecked,
				"listed", len(report.Discrepancies),
				"truncated", report.Truncated,
				"tb_compared", report.TBCompared,
			)
		} else {
			slog.Info("Ledger reconciliation clean",
				"tenant_id", tenant.ID,
				"invoices_checked", report.InvoicesChecked,
				"paid_invoices_checked", report.PaidInvoicesChecked,
			)
		}
	}
}
