package worker

import (
	"context"
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

func (w *CRMSyncWorker) Start() {
	w.ticker = time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-w.ticker.C:
				if _, err := w.RunOnce(context.Background()); err != nil {
					slog.Error("crm sync sweep failed", "error", err)
				}
			}
		}
	}()
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
		tctx := context.WithValue(ctx, domain.TenantIDKey, tenant.ID)
		// Resolve this tenant's CRM client (their own BYO account, else env).
		// A tenant with neither configured is skipped.
		crmClient := w.crmForTenant(tctx, tenant.ID)
		if crmClient == nil {
			continue
		}
		customers, err := w.customers.List(tctx, tenant.ID, domain.CustomerFilter{Limit: 10000})
		if err != nil {
			slog.Error("crm sync: customer list failed", "tenant_id", tenant.ID, "error", err)
			continue
		}
		active, err := w.subs.CountActiveByCustomer(tctx, tenant.ID)
		if err != nil {
			slog.Warn("crm sync: active counts unavailable", "tenant_id", tenant.ID, "error", err)
			active = map[uuid.UUID]int{}
		}
		for _, customer := range customers {
			if customer.Email == "" {
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
				continue
			}
			synced++
		}
	}
	if synced > 0 {
		slog.Info("crm sync sweep complete", "contacts", synced)
	}
	return synced, nil
}
