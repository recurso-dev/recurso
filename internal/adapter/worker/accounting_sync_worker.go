package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/swapnull-in/recur-so/internal/adapter/accounting"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

type AccountingSyncWorker struct {
	connRepo          port.AccountingConnectionRepository
	accountingService *service.AccountingService
	oauthConfigs      map[string]*accounting.OAuthConfig
	interval          time.Duration
}

func NewAccountingSyncWorker(
	connRepo port.AccountingConnectionRepository,
	accountingService *service.AccountingService,
	oauthConfigs map[string]*accounting.OAuthConfig,
	interval time.Duration,
) *AccountingSyncWorker {
	return &AccountingSyncWorker{
		connRepo:          connRepo,
		accountingService: accountingService,
		oauthConfigs:      oauthConfigs,
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
		// Refresh token if expired or about to expire
		if err := w.refreshTokenIfNeeded(ctx, conn); err != nil {
			slog.Error("token refresh failed", "tenant_id", conn.TenantID, "provider", conn.Provider, "error", err)
			conn.SyncStatus = "error"
			conn.LastError = "token refresh failed: " + err.Error()
			_ = w.connRepo.Update(ctx, conn)
			continue
		}

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

// refreshTokenIfNeeded checks if the OAuth token is expired and refreshes it
func (w *AccountingSyncWorker) refreshTokenIfNeeded(ctx context.Context, conn *domain.AccountingConnection) error {
	if conn.TokenExpiresAt == nil {
		return nil // No expiry info, assume valid
	}

	// Refresh if token expires within 5 minutes
	if time.Until(*conn.TokenExpiresAt) > 5*time.Minute {
		return nil // Token still valid
	}

	if conn.RefreshToken == "" {
		return nil // No refresh token available
	}

	slog.Info("refreshing OAuth token", "tenant_id", conn.TenantID, "provider", conn.Provider)

	config, ok := w.oauthConfigs[conn.Provider]
	if !ok {
		return fmt.Errorf("no OAuth config for provider %q", conn.Provider)
	}

	return accounting.RefreshAccessToken(ctx, config, conn, w.connRepo)
}
