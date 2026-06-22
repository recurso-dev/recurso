package gateway

import (
	"context"
	"fmt"

	"github.com/recur-so/recurso/internal/core/port"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
	"github.com/stripe/stripe-go/v76/webhook"
)

type StripeGateway struct {
	sc            *client.API
	webhookSecret string
}

func NewStripeGateway(apiKey string, webhookSecret string) *StripeGateway {
	return &StripeGateway{
		sc:            client.New(apiKey, nil),
		webhookSecret: webhookSecret,
	}
}

func (s *StripeGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: map[string]string{
			"receipt": receipt,
		},
	}

	pi, err := s.sc.PaymentIntents.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe create order failed: %v", err)
	}

	return &port.PaymentOrder{
		ID:       pi.ID,
		Amount:   pi.Amount,
		Currency: string(pi.Currency),
		Receipt:  receipt,
	}, nil
}

func (s *StripeGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	// If we have a webhook signature, we should verify it.
	// However, VerifyPayment signature in `port` is generic.
	// `paymentID` is the payload body in our generic handler usage?
	// Actually, looking at `SmartRouter`, it calls `VerifyPayment(ctx, orderID, paymentID, signature)`.
	// If this comes from a Webhook, `paymentID` might be the payload?
	// In razorpay: `g.VerifyPayment` uses orderID + paymentID + signature.

	// For Stripe, if this is called from a Webhook Handler, we expect the payload and headers.
	// But `VerifyPayment` signature `(orderID, paymentID, signature)` is tailored for Razorpay's client-side success callback?
	// If this is for server-side webhook, we usually pass the raw body.

	// Assuming this is used for verifying the "success" callback from the frontend (Client-side verification).
	// Stripe doesn't rely on client-side verification as trust. It relies on Webhooks.
	// But if we must support this method:
	// We can fetch the PaymentIntent from Stripe and check status.

	pi, err := s.sc.PaymentIntents.Get(orderID, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch stripe payment intent: %v", err)
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		return fmt.Errorf("payment intent status is %s", pi.Status)
	}

	return nil
}

func (s *StripeGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	// Need to find Customer by Email or Create
	// For simplicity, we create a new Customer every time or searching is better.
	// We'll search first.

	searchParams := &stripe.CustomerSearchParams{}
	searchParams.Query = fmt.Sprintf("email:'%s'", customerEmail)
	iter := s.sc.Customers.Search(searchParams)

	var customerID string
	if iter.Next() {
		customerID = iter.Current().(*stripe.Customer).ID
	} else {
		// Create
		cParams := &stripe.CustomerParams{
			Email: stripe.String(customerEmail),
		}
		c, err := s.sc.Customers.New(cParams)
		if err != nil {
			return "", fmt.Errorf("failed to create stripe customer: %v", err)
		}
		customerID = c.ID
	}

	// Create Subscription
	// We assume 'planID' passed here maps to a Stripe Price ID.
	// If not, we'd need a mapping capability.
	// We'll assert planID IS the Stripe Price ID (e.g. "price_123").

	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(planID),
			},
		},
	}
	// Stripe handles timestamps differently, usually immediate.
	// We'll ignore startAt for MVP unless critical.

	sub, err := s.sc.Subscriptions.New(subParams)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe subscription: %v", err)
	}

	return sub.ID, nil
}

func (s *StripeGateway) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, err := s.sc.Subscriptions.Cancel(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("stripe cancel subscription failed: %w", err)
	}
	return nil
}

func (s *StripeGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Confirm: stripe.Bool(true),
		Metadata: map[string]string{
			"invoice_id":   invoiceID,
			"retry_payment": "true",
		},
	}

	pi, err := s.sc.PaymentIntents.New(params)
	if err != nil {
		stripeErr, ok := err.(*stripe.Error)
		if ok {
			return &port.PaymentResult{
				Success:   false,
				ErrorCode: string(stripeErr.Code),
				ErrorMsg:  stripeErr.Msg,
			}, nil
		}
		return nil, fmt.Errorf("stripe retry payment infra error: %w", err)
	}

	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		return &port.PaymentResult{
			Success:   true,
			PaymentID: pi.ID,
		}, nil
	}

	return &port.PaymentResult{
		Success:   false,
		ErrorCode: string(pi.Status),
		ErrorMsg:  fmt.Sprintf("payment intent status: %s", pi.Status),
	}, nil
}

var ErrNotSupported = fmt.Errorf("operation not supported by this gateway")

func (s *StripeGateway) CreateMandate(ctx context.Context, customerEmail, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return nil, ErrNotSupported
}

func (s *StripeGateway) ExecuteMandateDebit(ctx context.Context, tokenID string, amount int64, currency, invoiceID string) (*port.PaymentResult, error) {
	return nil, ErrNotSupported
}

func (s *StripeGateway) RevokeMandate(ctx context.Context, tokenID string) error {
	return ErrNotSupported
}

func (s *StripeGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return nil, ErrNotSupported
}

// Helper for Webhook Handler to call directly if needed
func (s *StripeGateway) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, header, s.webhookSecret)
}
