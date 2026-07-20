package gateway

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// TenantGateway is a port.PaymentGateway that routes each call to the acting
// tenant's own gateway (BYO) when they have connected one, falling back to the
// env-configured gateway otherwise (spec D1). The tenant is read from the
// request context (domain.TenantIDKey); charge-origination sites that lack an
// authenticated tenant (public checkout/payment) inject the invoice's tenant id
// before calling.
//
// It is a thin dispatcher: swapping the injected paymentGateway for this
// wrapper leaves every service/handler call site unchanged, and with no vault
// or no connection the behavior is byte-for-byte the env gateway.
type TenantGateway struct {
	resolver *GatewayResolver
	env      port.PaymentGateway
}

func NewTenantGateway(resolver *GatewayResolver, env port.PaymentGateway) *TenantGateway {
	return &TenantGateway{resolver: resolver, env: env}
}

// route picks the gateway for the acting tenant, or the env gateway.
func (g *TenantGateway) route(ctx context.Context) port.PaymentGateway {
	if g.resolver == nil {
		return g.env
	}
	tid, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok || tid == uuid.Nil {
		return g.env
	}
	if r := g.resolver.For(ctx, tid); r != nil {
		return r
	}
	return g.env
}

func (g *TenantGateway) CreateOrder(ctx context.Context, amount int64, currency, receipt, invoiceID string) (*port.PaymentOrder, error) {
	return g.route(ctx).CreateOrder(ctx, amount, currency, receipt, invoiceID)
}

func (g *TenantGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	return g.route(ctx).VerifyPayment(ctx, orderID, paymentID, signature)
}

func (g *TenantGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return g.route(ctx).CreateSubscription(ctx, planID, totalCount, customerEmail, startAt, currency)
}

func (g *TenantGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	return g.route(ctx).RetryPayment(ctx, invoiceID, amount, currency)
}

func (g *TenantGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return g.route(ctx).CreateMandate(ctx, customerEmail, customerContact, vpa, maxAmount, frequency)
}

func (g *TenantGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	return g.route(ctx).ExecuteMandateDebit(ctx, req)
}

func (g *TenantGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	return g.route(ctx).RevokeMandate(ctx, customerID, tokenID)
}

func (g *TenantGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return g.route(ctx).CreateVirtualAccount(ctx, customerID, invoiceID, amount, description)
}

func (g *TenantGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return g.route(ctx).CancelSubscription(ctx, subscriptionID)
}

func (g *TenantGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	return g.route(ctx).Refund(ctx, paymentID, amount, currency)
}

// Compile-time guard: TenantGateway is a drop-in PaymentGateway.
var _ port.PaymentGateway = (*TenantGateway)(nil)
