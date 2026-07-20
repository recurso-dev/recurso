package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// stubGateway records its name on every order it creates so tests can assert
// which gateway a call routed to. Only the methods exercised here are
// meaningful; the rest satisfy the interface.
type stubGateway struct{ name string }

func (s *stubGateway) CreateOrder(_ context.Context, amount int64, currency, receipt, invoiceID string) (*port.PaymentOrder, error) {
	return &port.PaymentOrder{ID: "order_x", Amount: amount, Currency: currency, Gateway: s.name}, nil
}
func (s *stubGateway) VerifyPayment(context.Context, string, string, string) error { return nil }
func (s *stubGateway) CreateSubscription(context.Context, string, int, string, *int64, string) (string, error) {
	return "", nil
}
func (s *stubGateway) RetryPayment(context.Context, string, int64, string) (*port.PaymentResult, error) {
	return &port.PaymentResult{PaymentID: s.name}, nil
}
func (s *stubGateway) CreateMandate(context.Context, string, string, string, int64, string) (*port.MandateResult, error) {
	return &port.MandateResult{TokenID: s.name}, nil
}
func (s *stubGateway) ExecuteMandateDebit(context.Context, port.MandateDebitRequest) (*port.PaymentResult, error) {
	return &port.PaymentResult{PaymentID: s.name}, nil
}
func (s *stubGateway) RevokeMandate(context.Context, string, string) error { return nil }
func (s *stubGateway) CreateVirtualAccount(context.Context, string, string, int64, string) (*port.VirtualAccountResult, error) {
	return &port.VirtualAccountResult{VAID: s.name}, nil
}
func (s *stubGateway) CancelSubscription(context.Context, string) error { return nil }
func (s *stubGateway) Refund(context.Context, string, int64, string) (*port.RefundResult, error) {
	return &port.RefundResult{RefundID: s.name}, nil
}

// fakeVault is an in-memory ConnectionVault. Secrets are stored plaintext for
// the test; OpenSecret just returns them.
type fakeVault struct {
	conns map[uuid.UUID][]*domain.GatewayConnection
}

func (v *fakeVault) List(_ context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error) {
	return v.conns[tenantID], nil
}
func (v *fakeVault) OpenSecret(conn *domain.GatewayConnection) (string, error) {
	return conn.SecretKeyEnc, nil // plaintext in the fake
}

// newTestResolver wires a resolver whose env slots and per-tenant builds all
// produce named stubs, so routing is observable without any SDK/network.
func newTestResolver(vault ConnectionVault) *GatewayResolver {
	env := NewSmartRouter(&stubGateway{name: "env-razorpay"}, &stubGateway{name: "env-stripe"})
	r := NewGatewayResolver(vault, env)
	r.buildRazorpay = func(keyID, secret string) port.PaymentGateway { return &stubGateway{name: "byo-razorpay:" + secret} }
	r.buildStripe = func(secret string) port.PaymentGateway { return &stubGateway{name: "byo-stripe:" + secret} }
	return r
}

func orderGateway(t *testing.T, gw port.PaymentGateway, ctx context.Context, currency string) string {
	t.Helper()
	o, err := gw.CreateOrder(ctx, 100, currency, "rcpt", "inv")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	return o.Gateway
}

func TestTenantGatewayFallsBackWithoutTenant(t *testing.T) {
	vault := &fakeVault{conns: map[uuid.UUID][]*domain.GatewayConnection{}}
	tg := NewTenantGateway(newTestResolver(vault), &stubGateway{name: "env-direct"})

	// No tenant in ctx -> env fallback (the wrapper's own env, not the router).
	if got := orderGateway(t, tg, context.Background(), "USD"); got != "env-direct" {
		t.Fatalf("no-tenant: got %q want env-direct", got)
	}
}

func TestTenantGatewayFallsBackWhenNoConnection(t *testing.T) {
	tenant := uuid.New()
	vault := &fakeVault{conns: map[uuid.UUID][]*domain.GatewayConnection{}} // tenant has none
	resolver := newTestResolver(vault)
	tg := NewTenantGateway(resolver, resolver.env)

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenant)
	// Falls back to the env router: USD -> env-stripe, INR -> env-razorpay.
	if got := orderGateway(t, tg, ctx, "USD"); got != "env-stripe" {
		t.Fatalf("USD no-conn: got %q want env-stripe", got)
	}
	if got := orderGateway(t, tg, ctx, "INR"); got != "env-razorpay" {
		t.Fatalf("INR no-conn: got %q want env-razorpay", got)
	}
}

func TestTenantGatewayRoutesToTenantConnection(t *testing.T) {
	tenant := uuid.New()
	vault := &fakeVault{conns: map[uuid.UUID][]*domain.GatewayConnection{
		tenant: {
			{ID: uuid.New(), Provider: domain.GatewayStripe, PublicKey: "pk", SecretKeyEnc: "sksecret", Active: true, UpdatedAt: time.Unix(1, 0)},
		},
	}}
	resolver := newTestResolver(vault)
	tg := NewTenantGateway(resolver, resolver.env)
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenant)

	// USD routes to the tenant's own Stripe...
	if got := orderGateway(t, tg, ctx, "USD"); got != "byo-stripe:sksecret" {
		t.Fatalf("USD: got %q want byo-stripe:sksecret", got)
	}
	// ...but INR, which the tenant hasn't connected (no Razorpay), reuses the
	// env Razorpay slot.
	if got := orderGateway(t, tg, ctx, "INR"); got != "env-razorpay" {
		t.Fatalf("INR: got %q want env-razorpay (env fallback for un-connected slot)", got)
	}
}

func TestResolverCachesAndInvalidatesOnChange(t *testing.T) {
	tenant := uuid.New()
	conn := &domain.GatewayConnection{ID: uuid.New(), Provider: domain.GatewayStripe, SecretKeyEnc: "v1", Active: true, UpdatedAt: time.Unix(1, 0)}
	vault := &fakeVault{conns: map[uuid.UUID][]*domain.GatewayConnection{tenant: {conn}}}
	resolver := newTestResolver(vault)
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenant)

	r1 := resolver.For(ctx, tenant)
	r2 := resolver.For(ctx, tenant)
	if r1 != r2 {
		t.Fatal("expected the cached router to be reused for an unchanged connection set")
	}

	// Re-key (new secret + updated_at) must bust the cache and rebuild.
	conn.SecretKeyEnc = "v2"
	conn.UpdatedAt = time.Unix(2, 0)
	r3 := resolver.For(ctx, tenant)
	if r3 == r1 {
		t.Fatal("expected a rebuilt router after the connection changed")
	}
	tg := NewTenantGateway(resolver, resolver.env)
	if got := orderGateway(t, tg, ctx, "USD"); got != "byo-stripe:v2" {
		t.Fatalf("after re-key: got %q want byo-stripe:v2", got)
	}
}

func TestStripeForAndRazorpayFor(t *testing.T) {
	tenant := uuid.New()
	other := uuid.New()
	vault := &fakeVault{conns: map[uuid.UUID][]*domain.GatewayConnection{
		tenant: {
			{ID: uuid.New(), Provider: domain.GatewayStripe, SecretKeyEnc: "sk", Active: true, UpdatedAt: time.Unix(1, 0)},
		},
	}}
	resolver := newTestResolver(vault)
	ctx := context.Background()

	// Connected tenant -> their own Stripe; un-connected Razorpay slot -> env.
	if got := resolver.StripeFor(ctx, tenant).(*stubGateway).name; got != "byo-stripe:sk" {
		t.Fatalf("StripeFor(connected): got %q", got)
	}
	if got := resolver.RazorpayFor(ctx, tenant).(*stubGateway).name; got != "env-razorpay" {
		t.Fatalf("RazorpayFor(un-connected slot): got %q want env-razorpay", got)
	}
	// Tenant with no connections at all -> env for both.
	if got := resolver.StripeFor(ctx, other).(*stubGateway).name; got != "env-stripe" {
		t.Fatalf("StripeFor(no conn): got %q want env-stripe", got)
	}
}

func TestNilResolverAlwaysEnv(t *testing.T) {
	tg := NewTenantGateway(nil, &stubGateway{name: "env-direct"})
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, uuid.New())
	if got := orderGateway(t, tg, ctx, "USD"); got != "env-direct" {
		t.Fatalf("nil resolver: got %q want env-direct", got)
	}
}
