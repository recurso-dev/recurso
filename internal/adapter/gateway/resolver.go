package gateway

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// ConnectionVault is what the resolver needs from the gateway-connection
// service: the tenant's active connections and a way to open (decrypt) a
// connection's secret. Declared here (not imported from service) to keep the
// adapter layer free of a service dependency. *service.GatewayConnectionService
// satisfies it.
type ConnectionVault interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error)
	OpenSecret(conn *domain.GatewayConnection) (string, error)
}

// GatewayResolver builds a per-tenant SmartRouter from the tenant's stored
// (encrypted) gateway credentials, reusing the env-configured gateway for any
// provider the tenant hasn't connected. Results are cached per tenant and
// invalidated when the tenant's connection set changes (by id+updated_at
// signature), so charges don't re-decrypt on every call.
type GatewayResolver struct {
	vault ConnectionVault
	env   *SmartRouter // fallback per provider slot + currency overrides

	// Gateway constructors, injectable so tests can substitute stubs (the real
	// ones build live SDK clients).
	buildRazorpay func(keyID, secret string) port.PaymentGateway
	buildStripe   func(secret string) port.PaymentGateway

	mu    sync.RWMutex
	cache map[uuid.UUID]cachedRouter
}

type cachedRouter struct {
	sig    string
	router *SmartRouter
}

func NewGatewayResolver(vault ConnectionVault, env *SmartRouter) *GatewayResolver {
	return &GatewayResolver{
		vault:         vault,
		env:           env,
		buildRazorpay: func(keyID, secret string) port.PaymentGateway { return NewRazorpayGateway(keyID, secret) },
		// Webhook secret is used by the per-connection webhook router
		// (increment 3), not for outbound charges — empty here is fine.
		buildStripe: func(secret string) port.PaymentGateway { return NewStripeGateway(secret, "") },
		cache:       map[uuid.UUID]cachedRouter{},
	}
}

// For returns the tenant's gateway. When the tenant has no active connection
// (or the vault is unavailable), it returns nil so the caller falls back to the
// env gateway (D1). A non-nil router always covers both provider slots — the
// tenant's own gateway where connected, the env gateway otherwise.
func (r *GatewayResolver) For(ctx context.Context, tenantID uuid.UUID) *SmartRouter {
	if r == nil || r.vault == nil {
		return nil
	}
	conns, err := r.vault.List(ctx, tenantID)
	if err != nil || len(conns) == 0 {
		return nil
	}

	sig := connectionsSignature(conns)
	r.mu.RLock()
	if c, ok := r.cache[tenantID]; ok && c.sig == sig {
		r.mu.RUnlock()
		return c.router
	}
	r.mu.RUnlock()

	router := r.build(conns)
	r.mu.Lock()
	r.cache[tenantID] = cachedRouter{sig: sig, router: router}
	r.mu.Unlock()
	return router
}

// build assembles a SmartRouter: each connected provider uses the tenant's
// decrypted credentials; an un-connected slot falls back to the env gateway. A
// connection whose secret fails to open is skipped (falls back), never charged
// with a bad key.
func (r *GatewayResolver) build(conns []*domain.GatewayConnection) *SmartRouter {
	razorpay := r.env.Razorpay
	stripe := r.env.Stripe

	for _, conn := range conns {
		if !conn.HasSecret() {
			continue
		}
		secret, err := r.vault.OpenSecret(conn)
		if err != nil || secret == "" {
			continue
		}
		switch conn.Provider {
		case domain.GatewayRazorpay:
			razorpay = r.buildRazorpay(conn.PublicKey, secret)
		case domain.GatewayStripe:
			stripe = r.buildStripe(secret)
		}
	}

	router := NewSmartRouter(razorpay, stripe)
	// Preserve env currency overrides and extra gateways so BYO tenants keep
	// the same routing rules for currencies they haven't overridden.
	router.currencyOverrides = r.env.currencyOverrides
	router.Extra = r.env.Extra
	return router
}

// connectionsSignature is a stable fingerprint of a tenant's connection set;
// any change (new connection, re-key → new updated_at) busts the cache.
func connectionsSignature(conns []*domain.GatewayConnection) string {
	var b []byte
	for _, c := range conns {
		b = append(b, c.ID[:]...)
		b = append(b, []byte(c.UpdatedAt.Format("20060102150405.000000"))...)
	}
	return string(b)
}

// Ensure the resolver's product is a port.PaymentGateway (compile-time guard).
var _ port.PaymentGateway = (*SmartRouter)(nil)
