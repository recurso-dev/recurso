package service

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/service/tax"
)

// SalesTaxProviderResolver builds a tenant's own US sales-tax provider from
// their BYO integration connection (TaxJar or Avalara), caching the
// (rate-cache-wrapped) provider per tenant and rebuilding when the stored config
// changes. The raw provider is built by an injected factory so this package
// need not import the taxprovider adapter.
type SalesTaxProviderResolver struct {
	integrations *IntegrationConnectionService
	// build constructs a raw provider from a tenant's decrypted config. Returns
	// nil when the config can't produce one. Injected from main (has the adapter).
	build func(provider string, cfg map[string]string) tax.SalesTaxProvider

	mu    sync.RWMutex
	cache map[uuid.UUID]cachedSalesTax
}

type cachedSalesTax struct {
	sig      string
	provider tax.SalesTaxProvider
}

// NewSalesTaxProviderResolver wires the resolver. build turns a provider name +
// config into a raw tax.SalesTaxProvider (the resolver wraps it in the rate
// cache). A nil integrations service disables BYO (always returns nil → env).
func NewSalesTaxProviderResolver(integrations *IntegrationConnectionService, build func(provider string, cfg map[string]string) tax.SalesTaxProvider) *SalesTaxProviderResolver {
	return &SalesTaxProviderResolver{
		integrations: integrations,
		build:        build,
		cache:        map[uuid.UUID]cachedSalesTax{},
	}
}

// For returns the tenant's own sales-tax provider, or nil to fall back to the
// env provider. TaxJar is preferred over Avalara if both are somehow connected
// (the active-index makes that impossible per provider, not across providers).
func (r *SalesTaxProviderResolver) For(ctx context.Context, tenantID uuid.UUID) tax.SalesTaxProvider {
	if r == nil || r.integrations == nil || r.build == nil {
		return nil
	}
	for _, prov := range []string{"taxjar", "avalara"} {
		cfg, ok := r.integrations.Resolve(ctx, tenantID, domain.IntegrationTax, prov)
		if !ok {
			continue
		}
		sig := prov + "|" + configSignature(cfg)

		r.mu.RLock()
		if c, ok := r.cache[tenantID]; ok && c.sig == sig {
			r.mu.RUnlock()
			return c.provider
		}
		r.mu.RUnlock()

		raw := r.build(prov, cfg)
		if raw == nil {
			return nil
		}
		wrapped := tax.NewCachedSalesTaxProvider(raw, tax.DefaultSalesTaxRateTTL)
		r.mu.Lock()
		r.cache[tenantID] = cachedSalesTax{sig: sig, provider: wrapped}
		r.mu.Unlock()
		return wrapped
	}
	return nil
}

// configSignature is a stable fingerprint of a config map so a re-key busts the
// per-tenant cache.
func configSignature(cfg map[string]string) string {
	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(cfg[k])
		b.WriteByte(';')
	}
	return b.String()
}
