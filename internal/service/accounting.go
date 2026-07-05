package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/accounting"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// tokenRefreshWindow is how close to expiry a token may get before we
// proactively refresh it. QuickBooks access tokens live ~1h, Xero ~30min.
const tokenRefreshWindow = 5 * time.Minute

type AccountingService struct {
	gateway      port.AccountingGateway
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
	planRepo     port.PlanRepository
	connRepo     port.AccountingConnectionRepository
	oauthConfigs map[string]*accounting.OAuthConfig

	// adapterFactory overrides real adapter construction (used by tests).
	adapterFactory func(*domain.AccountingConnection) port.AccountingGateway
}

func NewAccountingService(
	gateway port.AccountingGateway,
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
) *AccountingService {
	return &AccountingService{
		gateway:      gateway,
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
		planRepo:     planRepo,
	}
}

func (s *AccountingService) SetConnectionRepo(repo port.AccountingConnectionRepository) {
	s.connRepo = repo
}

// SetOAuthConfigs provides the per-provider OAuth client credentials used to
// refresh expired connection tokens before syncing.
func (s *AccountingService) SetOAuthConfigs(configs map[string]*accounting.OAuthConfig) {
	s.oauthConfigs = configs
}

func (s *AccountingService) SyncCustomer(ctx context.Context, customerID uuid.UUID) error {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return err
	}
	return s.syncEntityAcrossConnections(ctx, customer.TenantID, "customer", customer.ID,
		func(gw port.AccountingGateway) error { return gw.SyncCustomer(ctx, customer) })
}

func (s *AccountingService) SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	return s.syncEntityAcrossConnections(ctx, invoice.TenantID, "invoice", invoice.ID,
		func(gw port.AccountingGateway) error { return gw.SyncInvoice(ctx, invoice) })
}

func (s *AccountingService) SyncProduct(ctx context.Context, planID string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return err
	}
	plan, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.syncEntityAcrossConnections(ctx, plan.TenantID, "product", plan.ID,
		func(gw port.AccountingGateway) error { return gw.SyncProduct(ctx, plan) })
}

// syncEntityAcrossConnections routes a single-entity sync through every
// active accounting connection for the tenant, refreshing OAuth tokens first.
func (s *AccountingService) syncEntityAcrossConnections(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID, sync func(port.AccountingGateway) error) error {
	if s.connRepo == nil {
		return fmt.Errorf("accounting connection repository not configured")
	}

	conns, err := s.connRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	var errs []error
	for _, conn := range conns {
		if !conn.IsActive {
			continue
		}

		if err := s.ensureFreshToken(ctx, conn); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", conn.Provider, err))
			continue
		}

		adapter := s.getAdapterForConnection(conn)
		if adapter == nil {
			continue
		}

		if err := sync(adapter); err != nil {
			s.logSyncResult(ctx, conn, entityType, entityID, "create", "error", err.Error())
			errs = append(errs, fmt.Errorf("%s: %w", conn.Provider, err))
			continue
		}
		s.logSyncResult(ctx, conn, entityType, entityID, "create", "success", "")
	}

	return errors.Join(errs...)
}

// ensureFreshToken refreshes the connection's OAuth token when it is expired
// or within tokenRefreshWindow of expiry. Rotated tokens are persisted before
// any sync uses them. On failure the connection is marked errored (and
// deactivated when the refresh token is definitively rejected) and an error
// is returned so callers skip the connection rather than sync with a dead
// token.
func (s *AccountingService) ensureFreshToken(ctx context.Context, conn *domain.AccountingConnection) error {
	if conn.TokenExpiresAt == nil {
		return nil // provider without token expiry (e.g. tally)
	}
	if time.Until(*conn.TokenExpiresAt) > tokenRefreshWindow {
		return nil // token still comfortably valid
	}

	if conn.RefreshToken == "" {
		return s.markConnectionError(ctx, conn, false,
			fmt.Errorf("access token expired and no refresh token available"))
	}

	config, ok := s.oauthConfigs[conn.Provider]
	if !ok || config == nil || config.ClientID == "" {
		return s.markConnectionError(ctx, conn, false,
			fmt.Errorf("no OAuth credentials configured for provider %q", conn.Provider))
	}

	if err := accounting.RefreshAccessToken(ctx, config, conn, s.connRepo); err != nil {
		return s.markConnectionError(ctx, conn, accounting.IsInvalidGrant(err), err)
	}

	slog.Info("refreshed accounting OAuth token", "connection_id", conn.ID, "provider", conn.Provider)
	return nil
}

// markConnectionError records a token failure on the connection. When the
// refresh token was definitively rejected (invalid_grant) the connection is
// deactivated so it is not retried until the merchant reconnects.
func (s *AccountingService) markConnectionError(ctx context.Context, conn *domain.AccountingConnection, deactivate bool, cause error) error {
	conn.SyncStatus = "error"
	conn.LastError = cause.Error()
	if deactivate {
		conn.IsActive = false
		conn.LastError = "refresh token rejected (invalid_grant); reconnect required: " + cause.Error()
	}
	if err := s.connRepo.Update(ctx, conn); err != nil {
		slog.Error("failed to persist connection error state", "connection_id", conn.ID, "error", err)
	}
	return cause
}

// SyncAllForTenant syncs all entities for a given tenant using the appropriate adapter.
func (s *AccountingService) SyncAllForTenant(ctx context.Context, tenantID uuid.UUID) error {
	if s.connRepo == nil {
		return fmt.Errorf("accounting connection repository not configured")
	}

	conns, err := s.connRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.IsActive {
			continue
		}

		if err := s.ensureFreshToken(ctx, conn); err != nil {
			slog.Error("skipping connection, token refresh failed",
				"connection_id", conn.ID, "provider", conn.Provider, "error", err)
			continue
		}

		adapter := s.getAdapterForConnection(conn)
		if adapter == nil {
			continue
		}

		// Sync customers (paginated)
		customerOffset := 0
		customerLimit := 100
		for {
			customers, err := s.customerRepo.List(ctx, tenantID, domain.CustomerFilter{Limit: customerLimit, Offset: customerOffset})
			if err != nil {
				slog.Error("failed to list customers for sync", "error", err)
				break
			}
			if len(customers) == 0 {
				break
			}

			for _, customer := range customers {
				if err := adapter.SyncCustomer(ctx, customer); err != nil {
					s.logSyncResult(ctx, conn, "customer", customer.ID, "create", "error", err.Error())
					continue
				}
				s.logSyncResult(ctx, conn, "customer", customer.ID, "create", "success", "")
			}

			if len(customers) < customerLimit {
				break
			}
			customerOffset += customerLimit
		}

		// Sync invoices
		invoices, err := s.invoiceRepo.List(ctx, tenantID)
		if err != nil {
			slog.Error("failed to list invoices for sync", "error", err)
			continue
		}

		for _, invoice := range invoices {
			if err := adapter.SyncInvoice(ctx, invoice); err != nil {
				s.logSyncResult(ctx, conn, "invoice", invoice.ID, "create", "error", err.Error())
				continue
			}
			s.logSyncResult(ctx, conn, "invoice", invoice.ID, "create", "success", "")
		}

		// Update connection status
		now := time.Now()
		conn.LastSyncAt = &now
		conn.SyncStatus = "synced"
		conn.LastError = ""
		_ = s.connRepo.Update(ctx, conn)
	}

	return nil
}

func (s *AccountingService) getAdapterForConnection(conn *domain.AccountingConnection) port.AccountingGateway {
	if s.adapterFactory != nil {
		return s.adapterFactory(conn)
	}

	switch conn.Provider {
	case "quickbooks":
		adapter := accounting.NewQuickBooksAdapter(conn.AccessToken, conn.RealmID, os.Getenv("QBO_SANDBOX") == "true")
		return adapter
	case "xero":
		adapter := accounting.NewXeroAdapter(conn.AccessToken, conn.RealmID)
		return adapter
	case "tally":
		adapter := accounting.NewTallyAdapter("")
		return adapter
	default:
		return s.gateway // Fall back to default (mock)
	}
}

func (s *AccountingService) logSyncResult(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, action, status, errMsg string) {
	if s.connRepo == nil {
		return
	}

	syncLog := &domain.AccountingSyncLog{
		ID:           uuid.New(),
		TenantID:     conn.TenantID,
		ConnectionID: conn.ID,
		EntityType:   entityType,
		EntityID:     entityID,
		Action:       action,
		Status:       status,
		ErrorMessage: errMsg,
		SyncedAt:     time.Now(),
	}

	if err := s.connRepo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("failed to create sync log", "error", err)
	}
}
