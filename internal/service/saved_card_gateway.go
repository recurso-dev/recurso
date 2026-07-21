package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// SavedCardCharger charges a saved payment method off-session. Satisfied by
// *gateway.StripeGateway. It is the capability the renewal / dunning / wallet
// paths need from a gateway.
type SavedCardCharger interface {
	ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
}

// gatewayConnectionOpener loads a stored gateway connection and decrypts its
// secret. Satisfied by *GatewayConnectionService.
type gatewayConnectionOpener interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.GatewayConnection, error)
	OpenSecret(conn *domain.GatewayConnection) (string, error)
}

// SavedCardGatewayRouter returns the off-session charger a saved card must be
// charged on (B1 autopay). A card's pm_* token is only valid on the gateway
// account that created it, so each charge routes to the connection the card was
// saved with:
//
//   - nil connection id  → the platform gateway (all pre-B1 cards and tenants
//     without a BYO connection);
//   - a connection id    → that tenant's BYO Stripe gateway, built from the
//     stored (decrypted) credentials.
//
// A connection that can't be loaded or opened is a hard error, not a silent
// fall back to the platform: charging a BYO-saved card on the platform account
// would always fail anyway, and the caller (renewal) leaves the invoice open for
// dunning, which prompts the customer to re-save.
type SavedCardGatewayRouter struct {
	conns       gatewayConnectionOpener
	buildStripe func(secret string) SavedCardCharger
	platform    SavedCardCharger
}

// NewSavedCardGatewayRouter builds the router. platform is the env/platform
// Stripe charger (used when a card has no recorded connection); buildStripe
// constructs a Stripe charger from a decrypted secret.
func NewSavedCardGatewayRouter(conns gatewayConnectionOpener, buildStripe func(secret string) SavedCardCharger, platform SavedCardCharger) *SavedCardGatewayRouter {
	return &SavedCardGatewayRouter{conns: conns, buildStripe: buildStripe, platform: platform}
}

// ChargerFor returns the charger for a saved card's gateway connection.
func (r *SavedCardGatewayRouter) ChargerFor(ctx context.Context, gatewayConnectionID *uuid.UUID) (SavedCardCharger, error) {
	if gatewayConnectionID == nil {
		return r.platform, nil
	}
	conn, err := r.conns.GetByID(ctx, *gatewayConnectionID)
	if err != nil {
		return nil, fmt.Errorf("load saved-card gateway connection %s: %w", *gatewayConnectionID, err)
	}
	if conn == nil {
		return nil, fmt.Errorf("saved-card gateway connection %s not found", *gatewayConnectionID)
	}
	if conn.Provider != domain.GatewayStripe {
		return nil, fmt.Errorf("saved-card gateway connection %s is %s, not stripe", *gatewayConnectionID, conn.Provider)
	}
	secret, err := r.conns.OpenSecret(conn)
	if err != nil || secret == "" {
		return nil, fmt.Errorf("open saved-card gateway secret for %s: %w", *gatewayConnectionID, err)
	}
	return r.buildStripe(secret), nil
}
