package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/recur-so/recurso/internal/core/port"
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

func (r *SmartRouter) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	if strings.ToUpper(currency) == "INR" {
		return r.Razorpay.CreateOrder(ctx, amount, currency, receipt)
	}
	return r.Stripe.CreateOrder(ctx, amount, currency, receipt)
}

func (r *SmartRouter) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	// This is tricky because we detect gateway by ID format
	if strings.HasPrefix(orderID, "order_") { // Razorpay Order IDs start with order_
		return r.Razorpay.VerifyPayment(ctx, orderID, paymentID, signature)
	}
	if strings.HasPrefix(orderID, "pi_") { // Stripe PaymentIntent
		return r.Stripe.VerifyPayment(ctx, orderID, paymentID, signature)
	}
	
	// Fallback/Error
	// Try Razorpay default if unsure? Or error.
	return fmt.Errorf("unknown gateway for order ID: %s", orderID)
}

func (r *SmartRouter) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	if strings.ToUpper(currency) == "INR" {
		return r.Razorpay.CreateSubscription(ctx, planID, totalCount, customerEmail, startAt, currency)
	}
	return r.Stripe.CreateSubscription(ctx, planID, totalCount, customerEmail, startAt, currency)
}

func (r *SmartRouter) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	if strings.ToUpper(currency) == "INR" {
		return r.Razorpay.RetryPayment(ctx, invoiceID, amount, currency)
	}
	return r.Stripe.RetryPayment(ctx, invoiceID, amount, currency)
}

// Mandate operations always route to Razorpay (UPI is India-only)
func (r *SmartRouter) CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return r.Razorpay.CreateMandate(ctx, customerEmail, vpa, maxAmount, frequency)
}

func (r *SmartRouter) ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	return r.Razorpay.ExecuteMandateDebit(ctx, tokenID, amount, currency, invoiceID)
}

func (r *SmartRouter) RevokeMandate(ctx context.Context, tokenID string) error {
	return r.Razorpay.RevokeMandate(ctx, tokenID)
}

// Virtual account operations route to Razorpay
func (r *SmartRouter) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return r.Razorpay.CreateVirtualAccount(ctx, customerID, invoiceID, amount, description)
}
