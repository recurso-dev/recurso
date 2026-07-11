package vault

import (
	"context"
	"fmt"
	"strconv"

	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
)

// StripeVault uses Stripe PaymentMethods API for card tokenization.
// Uses a per-instance client.API instead of the global stripe.Key to be
// safe for concurrent use and multiple Stripe accounts.
type StripeVault struct {
	sc *client.API
}

func NewStripeVault(apiKey string) *StripeVault {
	sc := client.New(apiKey, nil)
	return &StripeVault{sc: sc}
}

func (v *StripeVault) Tokenize(ctx context.Context, cardNumber, expMonth, expYear, cvc string) (*port.CardToken, error) {
	expMonthInt, _ := strconv.ParseInt(expMonth, 10, 64)
	expYearInt, _ := strconv.ParseInt(expYear, 10, 64)

	params := &stripe.PaymentMethodParams{
		Type: stripe.String(string(stripe.PaymentMethodTypeCard)),
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String(cardNumber),
			ExpMonth: stripe.Int64(expMonthInt),
			ExpYear:  stripe.Int64(expYearInt),
			CVC:      stripe.String(cvc),
		},
	}

	pm, err := v.sc.PaymentMethods.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe tokenization failed: %w", err)
	}

	return &port.CardToken{
		TokenID:     pm.ID,
		Fingerprint: pm.Card.Fingerprint,
		Brand:       string(pm.Card.Brand),
		Last4:       pm.Card.Last4,
		ExpMonth:    int(pm.Card.ExpMonth),
		ExpYear:     int(pm.Card.ExpYear),
	}, nil
}

func (v *StripeVault) Detokenize(ctx context.Context, tokenID string) (*port.CardToken, error) {
	pm, err := v.sc.PaymentMethods.Get(tokenID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe detokenize failed: %w", err)
	}

	return &port.CardToken{
		TokenID:     pm.ID,
		Fingerprint: pm.Card.Fingerprint,
		Brand:       string(pm.Card.Brand),
		Last4:       pm.Card.Last4,
		ExpMonth:    int(pm.Card.ExpMonth),
		ExpYear:     int(pm.Card.ExpYear),
	}, nil
}

func (v *StripeVault) DeleteToken(ctx context.Context, tokenID string) error {
	_, err := v.sc.PaymentMethods.Detach(tokenID, nil)
	if err != nil {
		return fmt.Errorf("stripe delete token failed: %w", err)
	}
	return nil
}

func (v *StripeVault) GetByFingerprint(ctx context.Context, fingerprint string) (*port.CardToken, error) {
	// Stripe doesn't support lookup by fingerprint directly.
	// This would require storing the fingerprint->tokenID mapping in our DB.
	return nil, nil
}
