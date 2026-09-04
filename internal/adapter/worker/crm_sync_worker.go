package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CRMSyncWorker pushes billing state into the CRM daily (Track D4): every
// customer becomes/updates a contact keyed by email, carrying the Recurso
// customer id and whether they hold an active subscription. Property
// writes are idempotent, so re-runs are safe.

// CRMContactUpserter is the CRM client slice; *crm.HubSpotClient. Exported so
// the per-tenant resolver (wired in main) can return one.
type CRMContactUpserter interface {
	UpsertContact(ctx context.Context, email string, properties map[string]string) (string, error)
}

// crmCustomerSource lists customers per tenant; *db.CustomerRepository.
type crmCustomerSource interface {
	List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
}

// crmSubscriptionCounter reports active-subscription counts by customer;
// *db.SubscriptionRepository.
type crmSubscriptionCounter interface {
	CountActiveByCustomer(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int, error)
}

type CRMSyncWorker struct {
	tenants   exportTenantLister
	customers crmCustomerSource
	subs      crmSubscriptionCounter
	crm       CRMContactUpserter // env/default client; may be nil when only BYO tenants exist
	// crmFor resolves a tenant's OWN CRM client (BYO), returning nil to fall
	// back to crm. Optional.
	crmFor   func(ctx context.Context, tenantID uuid.UUID) CRMContactUpserter
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
}

func NewCRMSyncWorker(tenants exportTenantLister, customers crmCustomerSource, subs crmSubscriptionCounter, crm CRMContactUpserter) *CRMSyncWorker {
	return &CRMSyncWorker{
		tenants:   tenants,
		customers: customers,
		subs:      subs,
		crm:       crm,
		interval:  24 * time.Hour,
		done:      make(chan bool),
	}
}

// SetPerTenantCRM wires a resolver returning a tenant's own (BYO) CRM client,
// used in preference to the env client. Returning nil falls back to the env
// client; a tenant with neither is skipped.
func (w *CRMSyncWorker) SetPerTenantCRM(fn func(ctx context.Context, tenantID uuid.UUID) CRMContactUpserter) {
	w.crmFor = fn
}

// crmForTenant picks the tenant's own client (BYO) when available, else env.
func (w *CRMSyncWorker) crmForTenant(ctx context.Context, tenantID uuid.UUID) CRMContactUpserter {
	if w.crmFor != nil {
		if c := w.crmFor(ctx, tenantID); c != nil {
			return c
		}
	}
	return w.crm
}

// Start runs the tick loop until ctx is cancelled or Stop is called. It is
// registered with main's worker group like every other worker, so shutdown
// cancels an in-flight sweep instead of letting it run to completion during
// drain (each tenant's RunOnce derives from ctx).
func (w *CRMSyncWorker) Start(ctx context.Context) {
	w.ticker = time.NewTicker(w.interval)
	defer w.ticker.Stop()
	w.announce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-w.ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				slog.ErrorContext(ctx, "crm sync sweep failed", "error", err)
			}
		}
	}
}

// announce logs the startup line once the loop is live.
func (w *CRMSyncWorker) announce() {
	slog.Info("crm sync worker started (daily)")
}

func (w *CRMSyncWorker) Stop() {
	w.stopOnce.Do(func() {
		if w.ticker != nil {
			w.ticker.Stop()
		}
		close(w.done)
		slog.Info("crm sync worker stopped")
	})
}

// RunOnce syncs every tenant's customers once; per-customer failures log
// and continue. Returns the number of contacts upserted.
func (w *CRMSyncWorker) RunOnce(ctx context.Context) (int, error) {
	tenants, err := w.tenants.ListTenants(ctx)
	if err != nil {
		return 0, fmt.Errorf("crm sync: list tenants: %w", err)
	}
	synced := 0
	for _, tenant := range tenants {
		n, _, err := w.RunTenant(ctx, tenant.ID, 0)
		if err != nil {
			slog.Error("crm sync: tenant sweep failed", "tenant_id", tenant.ID, "error", err)
			continue
		}
		synced += n
	}
	if synced > 0 {
		slog.Info("crm sync sweep complete", "contacts", synced)
	}
	return synced, nil
}

// ErrCRMNotConfigured marks a manual sync for a tenant with no CRM connected
// (neither BYO nor env). Handlers map it to a client error, not a 500.
var ErrCRMNotConfigured = errors.New("no CRM connected for this workspace")

// RunTenant syncs ONE tenant's customers to its CRM — the daily sweep's
// per-tenant body, exposed so the dashboard's "Sync now" can test a fresh
// connection without waiting for the 24h tick. maxContacts caps how many
// contacts are pushed this call (0 = unbounded, the daily sweep's mode):
// the manual sync must answer within proxy timeouts (Cloudflare kills the
// request at ~100s and the browser sees a bare failure), so it verifies the
// connection on a fast batch and reports what the sweep will finish.
// Returns (synced, remaining-eligible, err).
func (w *CRMSyncWorker) RunTenant(ctx context.Context, tenantID uuid.UUID, maxContacts int) (int, int, error) {
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	// Resolve this tenant's CRM client (their own BYO account, else env).
	crmClient := w.crmForTenant(tctx, tenantID)
	if crmClient == nil {
		return 0, 0, ErrCRMNotConfigured
	}
	customers, err := w.customers.List(tctx, tenantID, domain.CustomerFilter{Limit: 10000})
	if err != nil {
		return 0, 0, fmt.Errorf("crm sync: customer list: %w", err)
	}
	active, err := w.subs.CountActiveByCustomer(tctx, tenantID)
	if err != nil {
		slog.Warn("crm sync: active counts unavailable", "tenant_id", tenantID, "error", err)
		active = map[uuid.UUID]int{}
	}
	// Bootstrap the custom contact properties the sweep writes — HubSpot
	// rejects writes to undefined properties (PROPERTY_DOESNT_EXIST, seen
	// live). Optional capability: mocks and other CRMs skip it; a failure is
	// THE sync error (it names the missing scope, which is actionable).
	if bootstrapper, ok := crmClient.(interface{ EnsureProperties(context.Context) error }); ok {
		if err := bootstrapper.EnsureProperties(tctx); err != nil {
			return 0, 0, fmt.Errorf("crm property bootstrap: %w", err)
		}
	}
	synced, remaining := 0, 0
	var lastErr error
	for _, customer := range customers {
		if customer.Email == "" {
			continue
		}
		if maxContacts > 0 && synced >= maxContacts {
			remaining++
			continue
		}
		status := "churned"
		if active[customer.ID] > 0 {
			status = "active"
		}
		_, err := crmClient.UpsertContact(tctx, customer.Email, map[string]string{
			"recurso_customer_id":        customer.ID.String(),
			"recurso_subscription_state": status,
		})
		if err != nil {
			slog.Warn("crm sync: contact upsert failed", "customer_id", customer.ID, "error", err)
			lastErr = err
			continue
		}
		synced++
	}
	// A sync that upserted NOTHING and errored is a failed sync (bad token,
	// missing scopes) — surface the provider's error instead of "0 synced".
	if synced == 0 && lastErr != nil {
		return 0, 0, lastErr
	}
	return synced, remaining, nil
}
