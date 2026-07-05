package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

type SmartRouter struct {
	Razorpay port.PaymentGateway
	Stripe   port.PaymentGateway
}

func NewSmartRouter(razorpay port.PaymentGateway, stripe port.PaymentGateway) *SmartRouter {
	return &SmartRouter{
		Razorpay: razorpay,
		Stripe:   stripe,
	}
}

// gatewayFor returns the gateway for a currency, or an error when that
// gateway was not configured (previously a nil-pointer panic).
func (r *SmartRouter) gatewayFor(currency string) (port.PaymentGateway, error) {
	if strings.ToUpper(currency) == "INR" {
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

func (r *SmartRouter) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	gw, err := r.gatewayFor(currency)
	if err != nil {
		return nil, err
	}
	return gw.CreateOrder(ctx, amount, currency, receipt)
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
func (r *SmartRouter) CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return r.Razorpay.CreateMandate(ctx, customerEmail, vpa, maxAmount, frequency)
}

func (r *SmartRouter) ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	return r.Razorpay.ExecuteMandateDebit(ctx, tokenID, amount, currency, invoiceID)
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
