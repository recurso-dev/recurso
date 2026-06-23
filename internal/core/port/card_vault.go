package port

import "context"

// CardToken represents a tokenized card reference
type CardToken struct {
	TokenID     string // Vault-specific token (e.g., Stripe PaymentMethod ID)
	Fingerprint string // Card fingerprint for dedup
	Brand       string // visa, mastercard, amex
	Last4       string
	ExpMonth    int
	ExpYear     int
}

// CardVault provides secure card tokenization and storage.
// Implementations may use Stripe, a dedicated vault service, or in-memory for testing.
type CardVault interface {
	Tokenize(ctx context.Context, cardNumber, expMonth, expYear, cvc string) (*CardToken, error)
	Detokenize(ctx context.Context, tokenID string) (*CardToken, error)
	DeleteToken(ctx context.Context, tokenID string) error
	GetByFingerprint(ctx context.Context, fingerprint string) (*CardToken, error)
}
