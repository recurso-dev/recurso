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
	mappingRepo  port.AccountingMappingRepository
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

// SetMappingRepo provides the store for internal-to-external entity ID
// mappings. Without it, every sync re-runs the adapter create path (the
// adapters still dedupe by email/name where the provider allows a lookup).
func (s *AccountingService) SetMappingRepo(repo port.AccountingMappingRepository) {
	s.mappingRepo = repo
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
	return s.forEachActiveConnection(ctx, customer.TenantID,
		func(conn *domain.AccountingConnection, gw port.AccountingGateway) error {
			_, err := s.syncCustomerToConnection(ctx, conn, gw, customer, true)
			return err
		})
}

func (s *AccountingService) SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	return s.forEachActiveConnection(ctx, invoice.TenantID,
		func(conn *domain.AccountingConnection, gw port.AccountingGateway) error {
			return s.syncInvoiceToConnection(ctx, conn, gw, invoice)
		})
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
	return s.forEachActiveConnection(ctx, plan.TenantID,
		func(conn *domain.AccountingConnection, gw port.AccountingGateway) error {
			return s.syncProductToConnection(ctx, conn, gw, plan)
		})
}

// forEachActiveConnection routes work through every active accounting
// connection for the tenant, refreshing OAuth tokens first.
func (s *AccountingService) forEachActiveConnection(ctx context.Context, tenantID uuid.UUID, fn func(*domain.AccountingConnection, port.AccountingGateway) error) error {
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

		if err := fn(conn, adapter); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", conn.Provider, err))
		}
	}

	return errors.Join(errs...)
}

// syncCustomerToConnection ensures the customer exists on the provider's
// books and returns its provider-side ID. When a mapping already exists the
// adapter is not called again (create-once semantics; see the update gap
// notes on the adapters). logSkip controls whether the mapping-exists
// short-circuit is recorded in the sync log — explicit customer syncs log
// it, the implicit ensure-before-invoice path does not.
func (s *AccountingService) syncCustomerToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, customer *domain.Customer, logSkip bool) (string, error) {
	if extID, ok := s.lookupMapping(ctx, conn, "customer", customer.ID); ok {
		if logSkip {
			s.logSyncResult(ctx, conn, "customer", customer.ID, extID, "skip", "exists", "")
		}
		return extID, nil
	}

	extID, err := gw.SyncCustomer(ctx, customer)
	if err != nil {
		s.logSyncResult(ctx, conn, "customer", customer.ID, "", "create", "error", err.Error())
		return "", err
	}

	s.upsertMapping(ctx, conn, "customer", customer.ID, extID)
	s.logSyncResult(ctx, conn, "customer", customer.ID, extID, "create", "success", "")
	return extID, nil
}

// syncInvoiceToConnection syncs one invoice to one connection, first making
// sure the invoice's customer has a provider-side ID to reference.
func (s *AccountingService) syncInvoiceToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, invoice *domain.Invoice) error {
	if extID, ok := s.lookupMapping(ctx, conn, "invoice", invoice.ID); ok {
		// Already on the provider's books. Updates are not pushed (see
		// adapter update-gap notes); record the skip and move on.
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, extID, "skip", "exists", "")
		return nil
	}

	customer, err := s.customerRepo.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		err = fmt.Errorf("failed to load customer %s for invoice: %w", invoice.CustomerID, err)
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, "", "create", "error", err.Error())
		return err
	}

	customerExtID, err := s.syncCustomerToConnection(ctx, conn, gw, customer, false)
	if err != nil {
		err = fmt.Errorf("customer sync failed: %w", err)
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, "", "create", "error", err.Error())
		return err
	}

	extID, err := gw.SyncInvoice(ctx, invoice, customerExtID)
	if err != nil {
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, "", "create", "error", err.Error())
		return err
	}

	s.upsertMapping(ctx, conn, "invoice", invoice.ID, extID)
	s.logSyncResult(ctx, conn, "invoice", invoice.ID, extID, "create", "success", "")
	return nil
}

// syncProductToConnection syncs one plan to one connection with the same
// create-once semantics as customers.
func (s *AccountingService) syncProductToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, plan *domain.Plan) error {
	if extID, ok := s.lookupMapping(ctx, conn, "product", plan.ID); ok {
		s.logSyncResult(ctx, conn, "product", plan.ID, extID, "skip", "exists", "")
		return nil
	}

	extID, err := gw.SyncProduct(ctx, plan)
	if err != nil {
		s.logSyncResult(ctx, conn, "product", plan.ID, "", "create", "error", err.Error())
		return err
	}

	s.upsertMapping(ctx, conn, "product", plan.ID, extID)
	s.logSyncResult(ctx, conn, "product", plan.ID, extID, "create", "success", "")
	return nil
}

// lookupMapping returns the stored external ID for the entity on this
// connection, if any. A nil mapping repo behaves as "no mapping".
func (s *AccountingService) lookupMapping(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID) (string, bool) {
	if s.mappingRepo == nil {
		return "", false
	}
	m, err := s.mappingRepo.Get(ctx, conn.ID, entityType, entityID)
	if err != nil {
		slog.Error("failed to look up accounting mapping",
			"connection_id", conn.ID, "entity_type", entityType, "entity_id", entityID, "error", err)
		return "", false
	}
	if m == nil || m.ExternalID == "" {
		return "", false
	}
	return m.ExternalID, true
}

// upsertMapping records the provider-side ID returned by an adapter. Empty
// external IDs (e.g. providers without stable IDs) are not stored.
func (s *AccountingService) upsertMapping(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, externalID string) {
	if s.mappingRepo == nil || externalID == "" {
		return
	}
	m := &domain.AccountingEntityMapping{
		ID:           uuid.New(),
		TenantID:     conn.TenantID,
		ConnectionID: conn.ID,
		EntityType:   entityType,
		EntityID:     entityID,
		ExternalID:   externalID,
	}
	if err := s.mappingRepo.Upsert(ctx, m); err != nil {
		slog.Error("failed to upsert accounting mapping",
			"connection_id", conn.ID, "entity_type", entityType, "entity_id", entityID, "error", err)
	}
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
				_, _ = s.syncCustomerToConnection(ctx, conn, adapter, customer, true)
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
			_ = s.syncInvoiceToConnection(ctx, conn, adapter, invoice)
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

func (s *AccountingService) logSyncResult(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, externalID, action, status, errMsg string) {
	if s.connRepo == nil {
		return
	}

	syncLog := &domain.AccountingSyncLog{
		ID:           uuid.New(),
		TenantID:     conn.TenantID,
		ConnectionID: conn.ID,
		EntityType:   entityType,
		EntityID:     entityID,
		ExternalID:   externalID,
		Action:       action,
		Status:       status,
		ErrorMessage: errMsg,
		SyncedAt:     time.Now(),
	}

	if err := s.connRepo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("failed to create sync log", "error", err)
	}
}
