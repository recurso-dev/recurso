package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/port"
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

// stripePaymentMethodTypes returns the Stripe payment_method_types to enable on
// a PaymentIntent for the given ISO-4217 currency. Card is always offered;
// euro-denominated checkouts additionally surface European local methods
// (SEPA Direct Debit, iDEAL, Bancontact). This is the single source of truth
// for the currency -> payment-method mapping.
//
// IMPORTANT: every method returned here must ALSO be activated for the Stripe
// account in the Dashboard (Settings -> Payment methods). Passing a method that
// is not enabled there causes PaymentIntent creation to fail with an
// "The provided PaymentMethod type ... is invalid" error.
//
// Settlement timing (see docs):
//   - card, ideal, bancontact: authorize within seconds/minutes.
//   - sepa_debit: authorized immediately but funds settle over several business
//     days. The invoice is only marked paid on the payment_intent.succeeded
//     webhook, which fires once settlement completes.
//
// isInactivePaymentMethodErr reports whether Stripe rejected the
// PaymentIntent because a requested payment_method_type isn't activated on the
// account (error code payment_intent_invalid_parameter on that param) —
// verified live: accounts without ACH enabled reject us_bank_account this way.
func isInactivePaymentMethodErr(err error) bool {
	var se *stripe.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code == stripe.ErrorCode("payment_intent_invalid_parameter") && se.Param == "payment_method_types"
}

func stripePaymentMethodTypes(currency string) []string {
	switch strings.ToUpper(currency) {
	case "EUR":
		// iDEAL (NL), Bancontact (BE) and SEPA Direct Debit are all
		// euro-only local methods, so they are gated on EUR.
		return []string{
			string(stripe.PaymentMethodTypeCard),
			string(stripe.PaymentMethodTypeSEPADebit),
			string(stripe.PaymentMethodTypeIDEAL),
			string(stripe.PaymentMethodTypeBancontact),
		}
	case "USD":
		// ACH Direct Debit (us_bank_account) is the standard US B2B rail —
		// lower-fee than cards and common for recurring SaaS invoices. Like
		// SEPA it settles over a few business days (processing -> succeeded),
		// so it rides the same invoice_id-in-metadata async reconciliation as
		// SEPA. Card also carries Apple Pay / Google Pay wallets.
		return []string{
			string(stripe.PaymentMethodTypeCard),
			string(stripe.PaymentMethodTypeUSBankAccount),
		}
	default:
		// Card covers all remaining currencies (GBP, etc.) and also carries
		// Apple Pay / Google Pay wallets.
		return []string{string(stripe.PaymentMethodTypeCard)}
	}
}

func (s *StripeGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string, invoiceID string) (*port.PaymentOrder, error) {
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amount),
		Currency:           stripe.String(currency),
		PaymentMethodTypes: stripe.StringSlice(stripePaymentMethodTypes(currency)),
		// India-region Stripe accounts treat foreign-currency charges as
		// exports and reject confirmation without a description
		// (stripe.com/docs/india-exports); it also labels the payment in the
		// dashboard everywhere else.
		Description: stripe.String("Invoice " + receipt),
		// invoice_id lets the payment_intent.succeeded webhook reconcile
		// asynchronously-settling methods (SEPA) where there is no
		// synchronous checkout callback.
		Metadata: map[string]string{
			"receipt":    receipt,
			"invoice_id": invoiceID,
		},
	}

	pi, err := s.sc.PaymentIntents.New(params)
	if err != nil && isInactivePaymentMethodErr(err) {
		// One of the currency's extra method types (e.g. us_bank_account) isn't
		// activated on this Stripe account. A card-only checkout beats a dead
		// one — retry with card and let ops activate the method in the Stripe
		// dashboard when they want it.
		params.PaymentMethodTypes = stripe.StringSlice([]string{string(stripe.PaymentMethodTypeCard)})
		pi, err = s.sc.PaymentIntents.New(params)
	}
	if err != nil {
		return nil, fmt.Errorf("stripe create order failed: %v", err)
	}

	return &port.PaymentOrder{
		ID:       pi.ID,
		Amount:   pi.Amount,
		Currency: string(pi.Currency),
		Receipt:  receipt,
		// The client_secret lets the frontend Payment Element confirm this exact
		// PaymentIntent. It is safe to expose to the buyer's browser (it only
		// authorizes confirming this one intent), unlike the secret API key.
		ClientSecret: pi.ClientSecret,
		Gateway:      "stripe",
	}, nil
}

// SetOrderBuyer attaches the buyer's name and address to a PaymentIntent as
// shipping details. India-region Stripe accounts refuse to confirm
// foreign-currency charges without them (stripe.com/docs/india-exports);
// elsewhere it just labels the payment in the dashboard.
func (s *StripeGateway) SetOrderBuyer(ctx context.Context, orderID, name, line1, city, state, zip, country string) error {
	params := &stripe.PaymentIntentParams{
		Shipping: &stripe.ShippingDetailsParams{
			Name: stripe.String(name),
			Address: &stripe.AddressParams{
				Line1:      stripe.String(line1),
				City:       stripe.String(city),
				State:      stripe.String(state),
				PostalCode: stripe.String(zip),
				Country:    stripe.String(country),
			},
		},
	}
	_, err := s.sc.PaymentIntents.Update(orderID, params)
	return err
}

// GetPaymentStatus fetches a PaymentIntent so a checkout can be verified
// server-side before an invoice is marked paid. It returns the intent's status
// plus the invoice_id recorded in its metadata at CreateOrder time — the caller
// must confirm that invoice_id matches the invoice being settled, so a
// succeeded intent for one invoice can never be replayed to pay another.
func (s *StripeGateway) GetPaymentStatus(ctx context.Context, orderID string) (*port.PaymentStatus, error) {
	pi, err := s.sc.PaymentIntents.Get(orderID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get payment intent %s failed: %w", orderID, err)
	}
	return &port.PaymentStatus{
		Status:         string(pi.Status),
		InvoiceID:      pi.Metadata["invoice_id"],
		PaymentID:      pi.ID,
		AmountReceived: pi.AmountReceived,
	}, nil
}

// EnsureStripeCustomer returns existingID unchanged if set, otherwise creates a
// Stripe Customer for the given email/name and returns its id. Saved payment
// methods (from the portal SetupIntent flow) attach to this stable customer.
func (s *StripeGateway) EnsureStripeCustomer(ctx context.Context, existingID, email, name string) (string, error) {
	if existingID != "" {
		return existingID, nil
	}
	params := &stripe.CustomerParams{}
	if email != "" {
		params.Email = stripe.String(email)
	}
	if name != "" {
		params.Name = stripe.String(name)
	}
	params.Context = ctx
	c, err := s.sc.Customers.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create customer failed: %w", err)
	}
	return c.ID, nil
}

// CreateSetupIntent creates a SetupIntent to collect and save a reusable card
// against stripeCustomerID for future off-session charges, returning the
// client_secret the browser's Payment Element confirms. metadata carries the
// Recurso customer_id so the confirm endpoint can bind the saved method to the
// right customer — there is no setup_intent webhook handler; the portal
// finalizes inline or on redirect-return via /portal/api/payment-method/confirm.
// (Card only for now; ACH-mandate save is a later enhancement.)
func (s *StripeGateway) CreateSetupIntent(ctx context.Context, stripeCustomerID string, metadata map[string]string) (string, error) {
	params := &stripe.SetupIntentParams{
		Customer:           stripe.String(stripeCustomerID),
		Usage:              stripe.String("off_session"),
		PaymentMethodTypes: stripe.StringSlice([]string{string(stripe.PaymentMethodTypeCard)}),
	}
	for k, v := range metadata {
		params.AddMetadata(k, v)
	}
	params.Context = ctx
	si, err := s.sc.SetupIntents.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create setup intent failed: %w", err)
	}
	return si.ClientSecret, nil
}

// FinalizeSetupIntent reads back a confirmed SetupIntent, returning the saved
// payment method's card details and the Recurso customer_id from its metadata.
// On success it also sets the method as the Stripe Customer's default so future
// invoices charge it (best-effort). The caller must confirm CustomerID matches
// the authenticated portal customer before persisting anything.
func (s *StripeGateway) FinalizeSetupIntent(ctx context.Context, setupIntentID string) (*port.SavedCard, error) {
	si, err := s.sc.SetupIntents.Get(setupIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get setup intent %s failed: %w", setupIntentID, err)
	}

	out := &port.SavedCard{
		Status:     string(si.Status),
		CustomerID: si.Metadata["customer_id"],
	}
	if si.Status != stripe.SetupIntentStatusSucceeded || si.PaymentMethod == nil {
		return out, nil
	}
	out.PaymentMethodID = si.PaymentMethod.ID

	if pm, pmErr := s.sc.PaymentMethods.Get(si.PaymentMethod.ID, nil); pmErr == nil && pm.Card != nil {
		out.Brand = string(pm.Card.Brand)
		out.Last4 = pm.Card.Last4
		out.ExpMonth = int(pm.Card.ExpMonth)
		out.ExpYear = int(pm.Card.ExpYear)
	}

	// Make it the default for future (off-session) invoices. Best-effort — the
	// method is already saved even if this update fails.
	if si.Customer != nil {
		_, _ = s.sc.Customers.Update(si.Customer.ID, &stripe.CustomerParams{
			InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
				DefaultPaymentMethod: stripe.String(si.PaymentMethod.ID),
			},
		})
	}
	return out, nil
}

// ChargeSavedPaymentMethod charges a customer's saved payment method
// off-session (unattended, for recurring/retry collection) — the card was
// authorized earlier via the portal SetupIntent flow. idempotencyKey guards
// against double-charging the same retry attempt on a worker re-run. A card
// decline (or a required 3DS step, which off-session can't perform) is returned
// as a business failure so dunning handles it; only transport errors surface as
// an error.
func (s *StripeGateway) ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error) {
	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amount),
		Currency:      stripe.String(currency),
		Customer:      stripe.String(stripeCustomerID),
		PaymentMethod: stripe.String(paymentMethodID),
		OffSession:    stripe.Bool(true),
		Confirm:       stripe.Bool(true),
		// India-region Stripe accounts reject foreign-currency (export) charges
		// without a description (stripe.com/docs/india-exports) — same rule the
		// online checkout (CreateOrder) already satisfies. Without this, every
		// off-session dunning retry of a foreign-currency invoice fails.
		Description: stripe.String("Invoice " + invoiceID),
		Metadata: map[string]string{
			"invoice_id":    invoiceID,
			"retry_payment": "true",
		},
	}
	if idempotencyKey != "" {
		params.SetIdempotencyKey(idempotencyKey)
	}
	params.Context = ctx

	pi, err := s.sc.PaymentIntents.New(params)
	if err != nil {
		// Declines / authentication_required arrive as *stripe.Error — a dunning
		// failure, not an infra error.
		if stripeErr, ok := err.(*stripe.Error); ok {
			return &port.PaymentResult{
				Success:   false,
				ErrorCode: string(stripeErr.Code),
				ErrorMsg:  stripeErr.Msg,
			}, nil
		}
		return nil, fmt.Errorf("stripe off-session charge infra error: %w", err)
	}

	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		return &port.PaymentResult{Success: true, PaymentID: pi.ID}, nil
	}
	return &port.PaymentResult{
		Success:   false,
		ErrorCode: string(pi.Status),
		ErrorMsg:  fmt.Sprintf("payment intent status: %s", pi.Status),
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
			"invoice_id":    invoiceID,
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

func (s *StripeGateway) CreateMandate(ctx context.Context, customerEmail, customerContact, vpa string, maxAmount int64, frequency string) (*port.MandateResult, error) {
	return nil, ErrNotSupported
}

func (s *StripeGateway) ExecuteMandateDebit(ctx context.Context, req port.MandateDebitRequest) (*port.PaymentResult, error) {
	return nil, ErrNotSupported
}

func (s *StripeGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	return ErrNotSupported
}

func (s *StripeGateway) CreateVirtualAccount(ctx context.Context, customerID, invoiceID string, amount int64, description string) (*port.VirtualAccountResult, error) {
	return nil, ErrNotSupported
}

// Refund issues a (possibly partial) refund via the Stripe Refunds API.
// paymentID may be a PaymentIntent (pi_*) or a Charge (ch_*); currency is
// implied by the original payment, so the argument is unused here.
func (s *StripeGateway) Refund(ctx context.Context, paymentID string, amount int64, currency string) (*port.RefundResult, error) {
	params := &stripe.RefundParams{
		Amount: stripe.Int64(amount),
	}
	switch {
	case strings.HasPrefix(paymentID, "pi_"):
		params.PaymentIntent = stripe.String(paymentID)
	case strings.HasPrefix(paymentID, "ch_"):
		params.Charge = stripe.String(paymentID)
	default:
		return nil, fmt.Errorf("stripe refund: unrecognized payment id %q (expected pi_* or ch_*)", paymentID)
	}
	params.Context = ctx

	ref, err := s.sc.Refunds.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe refund failed for %s: %w", paymentID, err)
	}

	return &port.RefundResult{
		RefundID: ref.ID,
		Status:   string(ref.Status),
	}, nil
}

// Helper for Webhook Handler to call directly if needed
func (s *StripeGateway) ConstructEvent(payload []byte, header string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, header, s.webhookSecret)
}
