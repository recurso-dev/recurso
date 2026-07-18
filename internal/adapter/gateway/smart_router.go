package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/port"
)

type SmartRouter struct {
	Razorpay port.PaymentGateway
	Stripe   port.PaymentGateway
	// Extra holds additional gateways by name ("gocardless", "adyen", ...)
	// reachable only through currency overrides (Track D1).
	Extra map[string]port.PaymentGateway
	// currencyOverrides maps UPPER-CASE ISO currency -> gateway name,
	// consulted before the built-in INR->Razorpay / default->Stripe rule.
	currencyOverrides map[string]string
}

func NewSmartRouter(razorpay port.PaymentGateway, stripe port.PaymentGateway) *SmartRouter {
	return &SmartRouter{
		Razorpay: razorpay,
		Stripe:   stripe,
	}
}

// RegisterGateway makes an extra gateway addressable from currency
// overrides (Track D1).
func (r *SmartRouter) RegisterGateway(name string, gw port.PaymentGateway) {
	if r.Extra == nil {
		r.Extra = map[string]port.PaymentGateway{}
	}
	r.Extra[strings.ToLower(name)] = gw
}

// SetCurrencyOverrides parses "EUR=gocardless,GBP=gocardless,SGD=adyen"
// (GATEWAY_CURRENCY_OVERRIDES). Unknown gateway names error at boot rather
// than misrouting money at charge time.
func (r *SmartRouter) SetCurrencyOverrides(spec string) error {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	overrides := map[string]string{}
	for _, pair := range strings.Split(spec, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid GATEWAY_CURRENCY_OVERRIDES entry %q (want CUR=gateway)", pair)
		}
		name := strings.ToLower(strings.TrimSpace(parts[1]))
		if r.resolveByName(name) == nil {
			return fmt.Errorf("GATEWAY_CURRENCY_OVERRIDES names unconfigured gateway %q", name)
		}
		overrides[strings.ToUpper(strings.TrimSpace(parts[0]))] = name
	}
	r.currencyOverrides = overrides
	return nil
}

func (r *SmartRouter) resolveByName(name string) port.PaymentGateway {
	switch name {
	case "razorpay":
		return r.Razorpay
	case "stripe":
		return r.Stripe
	}
	return r.Extra[name]
}

// gatewayFor returns the gateway for a currency, or an error when that
// gateway was not configured (previously a nil-pointer panic).
func (r *SmartRouter) gatewayFor(currency string) (port.PaymentGateway, error) {
	cur := strings.ToUpper(currency)
	if name, ok := r.currencyOverrides[cur]; ok {
		if gw := r.resolveByName(name); gw != nil {
			return gw, nil
		}
		return nil, fmt.Errorf("gateway %q for currency %s not configured", name, cur)
	}
	if cur == "INR" {
		if r.Razorpay == nil {
			return nil, fmt.Errorf("razorpay gateway not configured for currency %s", currency)
		}
		return r.Razorpay, nil
	}
	if r.Stripe == nil {
		return nil, fmt.Errorf("stripe gateway not configured for currency %s", currency)
	}
	return r.Stripe, nil
}

func (r *SmartRouter) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	gw, err := r.gatewayFor(currency)
	if err != nil {
		return nil, err
	}
	return gw.CreateOrder(ctx, amount, currency, receipt, invoiceID)
}

func (r *SmartRouter) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	// This is tricky because we detect gateway by ID format
	if strings.HasPrefix(orderID, "order_") { // Razorpay Order IDs start with order_
		if r.Razorpay == nil {
			return fmt.Errorf("razorpay gateway not configured")
		}
		return r.Razorpay.VerifyPayment(ctx, orderID, paymentID, signature)
	}
	if strings.HasPrefix(orderID, "pi_") { // Stripe PaymentIntent
		if r.Stripe == nil {
			return fmt.Errorf("stripe gateway not configured")
		}
		return r.Stripe.VerifyPayment(ctx, orderID, paymentID, signature)
	}

	// Fallback/Error
	// Try Razorpay default if unsure? Or error.
	return fmt.Errorf("unknown gateway for order ID: %s", orderID)
}

func (r *SmartRouter) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	gw, err := r.gatewayFor(currency)
	if err != nil {
		return "", err
	}
	return gw.CreateSubscription(ctx, planID, totalCount, customerEmail, startAt, currency)
}

func (r *SmartRouter) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	gw, err := r.gatewayFor(currency)
	if err != nil {
		return nil, err
	}
	return gw.RetryPayment(ctx, invoiceID, amount, currency)
}

// Mandate operations always route to Razorpay (UPI is India-only)
func (r *SmartRouter) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return r.Razorpay.CreateMandate(ctx, customerEmail, customerContact, vpa, maxAmount, frequency)
}

func (r *SmartRouter) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	return r.Razorpay.ExecuteMandateDebit(ctx, req)
}

func (r *SmartRouter) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	return r.Razorpay.RevokeMandate(ctx, customerID, tokenID)
}

func (r *SmartRouter) CancelSubscription(ctx context.Context, subscriptionID string) error {
	// Both Razorpay and Stripe use "sub_" prefixes, so we cannot distinguish by ID alone.
	// The caller (subscription service) routes explicitly by gateway field.
	// Default to Stripe as it has a real implementation; Razorpay is a no-op.
	return r.Stripe.CancelSubscription(ctx, subscriptionID)
}

// Refund routes by the gateway payment id prefix (mirrors VerifyPayment's
// prefix routing): Stripe issues pi_/ch_ ids, Razorpay issues pay_ ids.
// Unknown prefixes (e.g. mock-era ids) fall back to currency routing.
func (r *SmartRouter) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	switch {
	case strings.HasPrefix(paymentID, "pi_") || strings.HasPrefix(paymentID, "ch_"):
		if r.Stripe == nil {
			return nil, fmt.Errorf("stripe gateway not configured for refund of %s", paymentID)
		}
		return r.Stripe.Refund(ctx, paymentID, amount, currency)
	case strings.HasPrefix(paymentID, "pay_"):
		if r.Razorpay == nil {
			return nil, fmt.Errorf("razorpay gateway not configured for refund of %s", paymentID)
		}
		return r.Razorpay.Refund(ctx, paymentID, amount, currency)
	}

	gw, err := r.gatewayFor(currency)
	if err != nil {
		return nil, err
	}
	return gw.Refund(ctx, paymentID, amount, currency)
}

// Virtual account operations route to Razorpay
func (r *SmartRouter) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return r.Razorpay.CreateVirtualAccount(ctx, customerID, invoiceID, amount, description)
}
