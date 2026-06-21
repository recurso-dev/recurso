package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
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

	for _, conn := range conns {
		slog.Info("syncing accounting", "tenant_id", conn.TenantID, "provider", conn.Provider)
		if err := w.accountingService.SyncAllForTenant(ctx, conn.TenantID); err != nil {
			slog.Error("accounting sync failed", "tenant_id", conn.TenantID, "error", err)

			conn.SyncStatus = "error"
			conn.LastError = err.Error()
			_ = w.connRepo.Update(ctx, conn)
			continue
		}

		now := time.Now()
		conn.SyncStatus = "synced"
		conn.LastSyncAt = &now
		conn.LastError = ""
		_ = w.connRepo.Update(ctx, conn)
	}

	slog.Info("accounting sync completed")
}
