package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/accounting"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/residency"
	"github.com/recurso-dev/recurso/internal/safego"
)

// tokenRefreshWindow is how close to expiry a token may get before we
// proactively refresh it. QuickBooks access tokens live ~1h, Xero ~30min.
const tokenRefreshWindow = 5 * time.Minute

type AccountingService struct {
	gateway          port.AccountingGateway
	customerRepo     port.CustomerRepository
	invoiceRepo      port.InvoiceRepository
	planRepo         port.PlanRepository
	subscriptionRepo port.SubscriptionRepository
	connRepo         port.AccountingConnectionRepository
	mappingRepo      port.AccountingMappingRepository
	oauthConfigs     map[string]*accounting.OAuthConfig

	// adapterFactory overrides real adapter construction (used by tests).
	adapterFactory func(*domain.AccountingConnection) port.AccountingGateway

	// syncInFlight guards manual syncs: one background full re-push per
	// tenant at a time (keys are tenant uuid.UUIDs).
	syncInFlight sync.Map
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

// SetSubscriptionRepo lets invoice syncs resolve the plan behind the
// invoice's subscription so the provider-side product/item can be referenced
// on invoice lines (QuickBooks ItemRef). Optional: without it invoice lines
// are synced with bare descriptions.
func (s *AccountingService) SetSubscriptionRepo(repo port.SubscriptionRepository) {
	s.subscriptionRepo = repo
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
			_, err := s.syncCustomerToConnection(ctx, conn, gw, customer)
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

// syncEntity is the shared create-or-update state machine for one entity on
// one connection. With no stored mapping the adapter is called in create mode
// (empty externalID); with one, in update mode. When the provider reports the
// mapped object gone (port.ErrExternalGone) the stale mapping is cleared and
// the entity is re-created once. The resulting provider-side ID is upserted
// as the mapping and the sync log records the action actually performed
// ("create" or "update").
func (s *AccountingService) syncEntity(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, call func(externalID string) (string, error)) (string, error) {
	extID, mapped := s.lookupMapping(ctx, conn, entityType, entityID)
	action := "create"
	if mapped {
		action = "update"
	}

	newExtID, err := call(extID)
	if mapped && errors.Is(err, port.ErrExternalGone) {
		s.clearStaleMapping(ctx, conn, entityType, entityID, extID)
		action = "create"
		newExtID, err = call("")
	}
	if err != nil {
		s.logSyncResult(ctx, conn, entityType, entityID, "", action, "error", err.Error())
		return "", err
	}

	s.upsertMapping(ctx, conn, entityType, entityID, newExtID)
	s.logSyncResult(ctx, conn, entityType, entityID, newExtID, action, "success", "")
	return newExtID, nil
}

// ensureEntityRef returns the provider-side ID for the entity, calling the
// adapter's create path only when no mapping exists yet. Unlike syncEntity it
// never pushes updates — it is the implicit dependency-resolution step (e.g.
// customer/product refs needed by an invoice), so a mapping hit is not
// logged.
func (s *AccountingService) ensureEntityRef(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, create func() (string, error)) (string, error) {
	if extID, ok := s.lookupMapping(ctx, conn, entityType, entityID); ok {
		return extID, nil
	}

	extID, err := create()
	if err != nil {
		s.logSyncResult(ctx, conn, entityType, entityID, "", "create", "error", err.Error())
		return "", err
	}

	s.upsertMapping(ctx, conn, entityType, entityID, extID)
	s.logSyncResult(ctx, conn, entityType, entityID, extID, "create", "success", "")
	return extID, nil
}

// syncCustomerToConnection pushes the customer to the provider — creating it
// when unmapped, updating the mapped object otherwise — and returns its
// provider-side ID.
func (s *AccountingService) syncCustomerToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, customer *domain.Customer) (string, error) {
	return s.syncEntity(ctx, conn, "customer", customer.ID, func(externalID string) (string, error) {
		return gw.SyncCustomer(ctx, customer, externalID)
	})
}

// syncInvoiceToConnection pushes one invoice to one connection, first making
// sure the invoice's customer (and, when the invoice is backed by a plan, the
// provider-side product/item) has a provider-side ID to reference.
func (s *AccountingService) syncInvoiceToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, invoice *domain.Invoice) error {
	customer, err := s.customerRepo.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		err = fmt.Errorf("failed to load customer %s for invoice: %w", invoice.CustomerID, err)
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, "", "create", "error", err.Error())
		return err
	}

	customerExtID, err := s.ensureEntityRef(ctx, conn, "customer", customer.ID, func() (string, error) {
		return gw.SyncCustomer(ctx, customer, "")
	})
	if err != nil {
		err = fmt.Errorf("customer sync failed: %w", err)
		s.logSyncResult(ctx, conn, "invoice", invoice.ID, "", "create", "error", err.Error())
		return err
	}

	refs := port.InvoiceSyncRefs{CustomerExternalID: customerExtID}
	productExtID, productCode, err := s.invoiceProductRef(ctx, conn, gw, invoice)
	if err != nil {
		// Non-fatal: the invoice still syncs with bare description lines.
		slog.Warn("could not resolve product ref for invoice; syncing with bare description lines",
			"invoice_id", invoice.ID, "connection_id", conn.ID, "error", err)
	} else {
		refs.ProductExternalID = productExtID
		refs.ProductCode = productCode
	}

	_, err = s.syncEntity(ctx, conn, "invoice", invoice.ID, func(externalID string) (string, error) {
		return gw.SyncInvoice(ctx, invoice, refs, externalID)
	})
	return err
}

// invoiceProductRef resolves the provider-side item ID and internal plan code
// for the plan backing the invoice's subscription, creating the item on the
// provider first if it has never been synced (same ensure-pattern as
// customers). The code is what Xero uses to link invoice lines to items
// (QuickBooks uses the ID). Returns empty values with no error when the
// invoice has no plan linkage (one-off invoice) or no subscription
// repository is wired.
func (s *AccountingService) invoiceProductRef(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, invoice *domain.Invoice) (extID, code string, err error) {
	if invoice.SubscriptionID == nil || s.subscriptionRepo == nil {
		return "", "", nil
	}

	sub, err := s.subscriptionRepo.GetByID(ctx, *invoice.SubscriptionID)
	if err != nil {
		return "", "", fmt.Errorf("failed to load subscription %s: %w", *invoice.SubscriptionID, err)
	}
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return "", "", fmt.Errorf("failed to load plan %s: %w", sub.PlanID, err)
	}

	extID, err = s.ensureEntityRef(ctx, conn, "product", plan.ID, func() (string, error) {
		return gw.SyncProduct(ctx, plan, "")
	})
	if err != nil {
		return "", "", err
	}
	return extID, plan.Code, nil
}

// syncProductToConnection pushes one plan to one connection with the same
// create-or-update semantics as customers.
func (s *AccountingService) syncProductToConnection(ctx context.Context, conn *domain.AccountingConnection, gw port.AccountingGateway, plan *domain.Plan) error {
	_, err := s.syncEntity(ctx, conn, "product", plan.ID, func(externalID string) (string, error) {
		return gw.SyncProduct(ctx, plan, externalID)
	})
	return err
}

// clearStaleMapping removes a mapping whose external ID the provider no
// longer recognizes, so the entity can be re-created cleanly.
func (s *AccountingService) clearStaleMapping(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, externalID string) {
	slog.Info("external accounting entity gone at provider; clearing mapping and re-creating",
		"connection_id", conn.ID, "entity_type", entityType, "entity_id", entityID, "external_id", externalID)
	if s.mappingRepo == nil {
		return
	}
	if err := s.mappingRepo.Delete(ctx, conn.ID, entityType, entityID); err != nil {
		slog.Error("failed to delete stale accounting mapping",
			"connection_id", conn.ID, "entity_type", entityType, "entity_id", entityID, "error", err)
	}
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

// SyncAllForTenant syncs all entities for a given tenant using the
// appropriate adapter. Entities that have not changed since their last
// successful sync (source updated_at not after the mapping's last-synced
// timestamp) are skipped unless force is set — the manual sync endpoint
// forces a full re-push, the daily worker does not.
// TriggerSyncAsync starts a forced full re-push for the tenant in the
// background and reports whether it started. A false return means a manual
// sync is already running for this tenant. Rationale: the sweep is O(all
// customers + invoices) against a third-party API (~85s observed on a small
// production tenant) while Cloudflare kills proxied requests around 100s —
// the same failure mode the CRM manual sync hit. Progress is observable via
// the sync-activity log and each connection's sync_status.
// provider "" sweeps every active connection; a provider name scopes the
// sweep to that one connection. Single-flight stays PER TENANT regardless of
// scope: a provider-scoped sync running concurrently with an all-provider
// sync would double-push the overlapping connection. force bypasses
// dirty-tracking (full re-push): right for the explicit reconcile-everything
// button, wildly wasteful for a routine card-level sync — a forced full
// re-push (~3 provider calls per invoice) exceeded the 15-minute budget on a
// real tenant, while an incremental pass skips unchanged records and
// finishes in seconds.
func (s *AccountingService) TriggerSyncAsync(tenantID uuid.UUID, provider string, force bool) bool {
	if _, running := s.syncInFlight.LoadOrStore(tenantID, struct{}{}); running {
		return false
	}
	safego.Go("accounting.TriggerSyncAsync", func() {
		defer s.syncInFlight.Delete(tenantID)
		// Detached from the HTTP request: the client navigating away must not
		// cancel a half-done books push. Bounded so a wedged provider cannot
		// hold the single-flight slot forever.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if err := s.syncForTenant(ctx, tenantID, force, provider); err != nil {
			slog.Error("manual accounting sync failed", "tenant_id", tenantID, "provider", provider, "error", err)
		}
	})
	return true
}

func (s *AccountingService) SyncAllForTenant(ctx context.Context, tenantID uuid.UUID, force bool) error {
	return s.syncForTenant(ctx, tenantID, force, "")
}

// syncForTenant is SyncAllForTenant optionally scoped to one provider.
func (s *AccountingService) syncForTenant(ctx context.Context, tenantID uuid.UUID, force bool, provider string) error {
	if s.connRepo == nil {
		return fmt.Errorf("accounting connection repository not configured")
	}

	// The sync worker calls with a background context, and the per-entity
	// helpers below read customers/subscriptions/plans through tenant-scoped
	// repos (tenant-context bug class) — inject the tenant being synced.
	ctx = context.WithValue(ctx, domain.TenantIDKey, tenantID)

	conns, err := s.connRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.IsActive {
			continue
		}
		if provider != "" && conn.Provider != provider {
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

		// Surface progress on the connection row while the sweep runs.
		conn.SyncStatus = "syncing"
		_ = s.connRepo.Update(ctx, conn)

		var synced, skipped int

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
				if ctx.Err() != nil {
					break // sweep budget exhausted: stop, don't spray dead calls
				}
				if !force && s.unchangedSinceLastSync(ctx, conn, "customer", customer.ID, customer.UpdatedAt) {
					skipped++
					continue
				}
				_, _ = s.syncCustomerToConnection(ctx, conn, adapter, customer)
				synced++
			}

			if len(customers) < customerLimit {
				break
			}
			customerOffset += customerLimit
		}

		// Sync invoices
		var legErr error
		invoices, err := s.invoiceRepo.List(ctx, tenantID)
		if err != nil {
			slog.Error("failed to list invoices for sync", "error", err)
			legErr = fmt.Errorf("list invoices: %w", err)
		} else {
			for _, invoice := range invoices {
				if ctx.Err() != nil {
					break
				}
				if !force && s.unchangedSinceLastSync(ctx, conn, "invoice", invoice.ID, invoice.UpdatedAt) {
					skipped++
					continue
				}
				_ = s.syncInvoiceToConnection(ctx, conn, adapter, invoice)
				synced++
			}
		}
		if legErr == nil && ctx.Err() != nil {
			legErr = fmt.Errorf("sync aborted: %w", ctx.Err())
		}

		s.finishSyncLeg(conn, tenantID, force, synced, skipped, legErr)
	}

	return nil
}

// finishSyncLeg records the terminal state of one connection's sweep leg. It
// writes with its OWN short-lived context: the sweep's context may already be
// dead (async syncs run under a 15-minute cap), and a leg that dies mid-flight
// must never leave sync_status stuck on "syncing" — observed live: a capped
// sweep left the row "syncing" forever, which read as an eternal in-flight
// sync on the dashboard.
func (s *AccountingService) finishSyncLeg(conn *domain.AccountingConnection, tenantID uuid.UUID, force bool, synced, skipped int, legErr error) {
	wctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	conn.LastSyncAt = &now
	if legErr != nil {
		conn.SyncStatus = "error"
		conn.LastError = legErr.Error()
		slog.Warn("accounting sync leg ended with error",
			"connection_id", conn.ID, "provider", conn.Provider, "tenant_id", tenantID,
			"force", force, "synced", synced, "skipped_unchanged", skipped, "error", legErr)
	} else {
		conn.SyncStatus = "synced"
		conn.LastError = ""
		slog.Info("accounting sync completed for connection",
			"connection_id", conn.ID, "provider", conn.Provider, "tenant_id", tenantID,
			"force", force, "synced", synced, "skipped_unchanged", skipped)
	}
	if err := s.connRepo.Update(wctx, conn); err != nil {
		slog.Error("failed to record sync terminal status", "connection_id", conn.ID, "error", err)
	}
}

// unchangedSinceLastSync reports whether the entity can be skipped on a bulk
// sync: it already has a mapping on this connection and its source updated_at
// is not after the mapping's updated_at (which every successful sync
// refreshes, making it the last-synced timestamp). Anything uncertain — no
// mapping repo, a zero source updated_at (row predating the updated_at
// column or a repo path that does not scan it), or a mapping lookup failure —
// reports false so the entity is synced rather than silently dropped.
func (s *AccountingService) unchangedSinceLastSync(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, sourceUpdatedAt time.Time) bool {
	if s.mappingRepo == nil || sourceUpdatedAt.IsZero() {
		return false
	}
	m, err := s.mappingRepo.Get(ctx, conn.ID, entityType, entityID)
	if err != nil {
		slog.Error("failed to look up accounting mapping for dirty check; syncing entity",
			"connection_id", conn.ID, "entity_type", entityType, "entity_id", entityID, "error", err)
		return false
	}
	if m == nil || m.ExternalID == "" {
		return false // never synced to this connection
	}
	return !sourceUpdatedAt.After(m.UpdatedAt)
}

func (s *AccountingService) getAdapterForConnection(conn *domain.AccountingConnection) port.AccountingGateway {
	if s.adapterFactory != nil {
		return s.adapterFactory(conn)
	}

	// Residency guarantee (docs/india-data-residency.md): under
	// RESIDENCY_MODE=self_hosted no financial data may leave for an
	// accounting SaaS, even when a connection row already exists in the
	// database. Tally stays available — it is a local file export, not
	// network egress.
	if residency.SelfHosted() && (conn.Provider == "quickbooks" || conn.Provider == "xero" || conn.Provider == "netsuite") {
		slog.Warn("accounting sync blocked by RESIDENCY_MODE=self_hosted",
			"provider", conn.Provider, "connection_id", conn.ID)
		return s.gateway // mock fallback: records nothing externally
	}

	switch conn.Provider {
	case "quickbooks":
		adapter := accounting.NewQuickBooksAdapter(conn.AccessToken, conn.RealmID, os.Getenv("QBO_SANDBOX") == "true")
		return adapter
	case "xero":
		adapter := accounting.NewXeroAdapter(conn.AccessToken, conn.RealmID)
		return adapter
	case "netsuite":
		// Track D2 (EXPERIMENTAL): SuiteTalk REST; RealmID carries the
		// NetSuite account id, mirroring how QBO uses it.
		adapter := accounting.NewNetSuiteAdapter(conn.AccessToken, conn.RealmID)
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
