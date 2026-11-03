package handler

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// orderAutopayGateway is the Stripe capability for keeping the card a checkout
// pays with. Satisfied by *gateway.StripeGateway; reached by type assertion so
// other gateways simply don't save.
type orderAutopayGateway interface {
	EnsureStripeCustomer(ctx context.Context, existingID, email, name string) (string, error)
	PrepareOrderForAutopay(ctx context.Context, orderID, stripeCustomerID string) error
	SavedCardFromOrder(ctx context.Context, orderID string) (*port.SavedCard, error)
}

// autopayCustomerStore extends the card store with reading the saved method.
// Satisfied by *db.CustomerRepository.
type autopayCustomerStore interface {
	customerPaymentStore
	GetSavedPaymentMethod(ctx context.Context, customerID uuid.UUID) (stripeCustomerID, paymentMethodID string, gatewayConnectionID *uuid.UUID, err error)
}

// CheckoutAutopay keeps the card a customer pays a subscription invoice with, so
// renewals can charge it. Without this a checkout payment leaves nothing on file
// and every renewal waits for the customer to pay by hand. Best-effort
// throughout: a failure here must never block or undo a payment.
type CheckoutAutopay struct {
	store    autopayCustomerStore
	resolver paymentSetupResolver
}

// NewCheckoutAutopay wires card saving; resolver picks the tenant's gateway the
// same way the portal's card flow does.
func NewCheckoutAutopay(store autopayCustomerStore, resolver paymentSetupResolver) *CheckoutAutopay {
	return &CheckoutAutopay{store: store, resolver: resolver}
}

func (a *CheckoutAutopay) gatewayFor(ctx context.Context, tenantID uuid.UUID) (orderAutopayGateway, *uuid.UUID) {
	setup, connID, _ := a.resolver.SetupForTenant(ctx, tenantID)
	gw, ok := setup.(orderAutopayGateway)
	if !ok {
		return nil, nil
	}
	return gw, connID
}

// Prepare marks a subscription invoice's order to save its card, and reports
// whether it did so the checkout page can tell the customer. A customer who
// already has a saved method keeps it.
func (a *CheckoutAutopay) Prepare(ctx context.Context, inv *domain.Invoice, orderID string) bool {
	if a == nil || inv.SubscriptionID == nil {
		return false
	}
	gw, _ := a.gatewayFor(ctx, inv.TenantID)
	if gw == nil {
		return false
	}
	existing, methodID, _, err := a.store.GetSavedPaymentMethod(ctx, inv.CustomerID)
	if err != nil || methodID != "" {
		return false
	}
	cust, err := a.store.GetByIDPublic(ctx, inv.CustomerID)
	if err != nil || cust == nil {
		slog.WarnContext(ctx, "checkout autopay: customer lookup failed", "invoice_id", inv.ID, "error", err)
		return false
	}
	stripeCustomerID, err := gw.EnsureStripeCustomer(ctx, existing, cust.Email, domain.PtrToString(cust.Name))
	if err != nil {
		slog.WarnContext(ctx, "checkout autopay: stripe customer failed", "invoice_id", inv.ID, "error", err)
		return false
	}
	if stripeCustomerID != existing {
		if err := a.store.SetStripeCustomerID(ctx, inv.CustomerID, stripeCustomerID); err != nil {
			slog.WarnContext(ctx, "checkout autopay: saving stripe customer failed", "invoice_id", inv.ID, "error", err)
			return false
		}
	}
	if err := gw.PrepareOrderForAutopay(ctx, orderID, stripeCustomerID); err != nil {
		slog.WarnContext(ctx, "checkout autopay: preparing order failed", "invoice_id", inv.ID, "error", err)
		return false
	}
	return true
}

// Save records the card a prepared order paid with. Both settlement paths call
// it, so it's safe to repeat, and a method saved in the meantime is kept.
func (a *CheckoutAutopay) Save(ctx context.Context, inv *domain.Invoice, orderID string) {
	if a == nil || inv.SubscriptionID == nil || orderID == "" {
		return
	}
	gw, connID := a.gatewayFor(ctx, inv.TenantID)
	if gw == nil {
		return
	}
	stripeCustomerID, methodID, _, err := a.store.GetSavedPaymentMethod(ctx, inv.CustomerID)
	if err != nil || methodID != "" {
		return
	}
	card, err := gw.SavedCardFromOrder(ctx, orderID)
	if err != nil {
		slog.WarnContext(ctx, "checkout autopay: reading order failed", "invoice_id", inv.ID, "error", err)
		return
	}
	// Bind to this customer: the card only ever belongs to the Stripe customer
	// Prepare attached to the order.
	if card == nil || card.CustomerID == "" || card.CustomerID != stripeCustomerID {
		return
	}
	if err := a.store.SetDefaultPaymentMethod(ctx, inv.CustomerID, card.PaymentMethodID, card.Brand, card.Last4, card.ExpMonth, card.ExpYear, connID); err != nil {
		slog.ErrorContext(ctx, "checkout autopay: saving card failed", "invoice_id", inv.ID, "error", err)
	}
}
