package worker

import (
	"context"
	"log"
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

	log.Println("Accounting sync worker started")

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
	log.Println("Running accounting sync for all active connections...")

	conns, err := w.connRepo.GetActiveConnections(ctx)
	if err != nil {
		log.Printf("Failed to get active accounting connections: %v", err)
		return
	}

	for _, conn := range conns {
		log.Printf("Syncing accounting for tenant %s via %s", conn.TenantID, conn.Provider)
		if err := w.accountingService.SyncAllForTenant(ctx, conn.TenantID); err != nil {
			log.Printf("Accounting sync failed for tenant %s: %v", conn.TenantID, err)

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

	log.Println("Accounting sync completed")
}
