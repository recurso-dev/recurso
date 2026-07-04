package vault

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/recur-so/recurso/internal/core/port"
)

// MockVault provides an in-memory card token store for dev/testing.
type MockVault struct {
	mu     sync.RWMutex
	tokens map[string]*port.CardToken
}

func NewMockVault() *MockVault {
	return &MockVault{
		tokens: make(map[string]*port.CardToken),
	}
}

func (v *MockVault) Tokenize(ctx context.Context, cardNumber, expMonth, expYear, cvc string) (*port.CardToken, error) {
	fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(cardNumber)))[:16]
	tokenID := fmt.Sprintf("tok_mock_%s", fingerprint[:8])

	// Determine brand from first digit
	brand := "unknown"
	if len(cardNumber) > 0 {
		switch cardNumber[0] {
		case '4':
			brand = "visa"
		case '5':
			brand = "mastercard"
		case '3':
			brand = "amex"
		}
	}

	last4 := ""
	if len(cardNumber) >= 4 {
		last4 = cardNumber[len(cardNumber)-4:]
	}

	token := &port.CardToken{
		TokenID:     tokenID,
		Fingerprint: fingerprint,
		Brand:       brand,
		Last4:       last4,
	}

	// Parse exp month/year (best-effort; zero values on parse failure)
	_, _ = fmt.Sscanf(expMonth, "%d", &token.ExpMonth)
	_, _ = fmt.Sscanf(expYear, "%d", &token.ExpYear)

	v.mu.Lock()
	v.tokens[tokenID] = token
	v.mu.Unlock()

	return token, nil
}

func (v *MockVault) Detokenize(ctx context.Context, tokenID string) (*port.CardToken, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	token, ok := v.tokens[tokenID]
	if !ok {
		return nil, fmt.Errorf("token not found: %s", tokenID)
	}
	return token, nil
}

func (v *MockVault) DeleteToken(ctx context.Context, tokenID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.tokens, tokenID)
	return nil
}

func (v *MockVault) GetByFingerprint(ctx context.Context, fingerprint string) (*port.CardToken, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, token := range v.tokens {
		if token.Fingerprint == fingerprint {
			return token, nil
		}
	}
	return nil, nil
}
