package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

type AccountingSyncWorker struct {
	connRepo          port.AccountingConnectionRepository
	accountingService *service.AccountingService
	interval          time.Duration
}

func NewAccountingSyncWorker(
	connRepo port.AccountingConnectionRepository,
	accountingService *service.AccountingService,
	interval time.Duration,
) *AccountingSyncWorker {
	return &AccountingSyncWorker{
		connRepo:          connRepo,
		accountingService: accountingService,
		interval:          interval,
	}
}

func (w *AccountingSyncWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("accounting sync worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunSync(ctx)
		}
	}
}

func (w *AccountingSyncWorker) RunSync(ctx context.Context) {
	slog.Info("running accounting sync for all active connections")

	conns, err := w.connRepo.GetActiveConnections(ctx)
	if err != nil {
		slog.Error("failed to get active accounting connections", "error", err)
		return
	}

	// SyncAllForTenant refreshes tokens per connection (deactivating dead
	// ones) and records per-connection status, so the worker only fans out
	// once per distinct tenant.
	synced := make(map[uuid.UUID]bool)
	for _, conn := range conns {
		if synced[conn.TenantID] {
			continue
		}
		synced[conn.TenantID] = true

		slog.Info("syncing accounting", "tenant_id", conn.TenantID)
		// Scheduled runs are incremental (force=false): entities unchanged
		// since their last successful sync are skipped.
		if err := w.accountingService.SyncAllForTenant(ctx, conn.TenantID, false); err != nil {
			slog.Error("accounting sync failed", "tenant_id", conn.TenantID, "error", err)
		}
	}

	slog.Info("accounting sync completed")
}
