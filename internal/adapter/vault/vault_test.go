package vault_test

import (
	"context"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/vault"
)

func TestMockVault_TokenizeAndDetokenize(t *testing.T) {
	v := vault.NewMockVault()
	ctx := context.Background()

	// Tokenize Visa
	tok, err := v.Tokenize(ctx, "4242424242424242", "12", "2028", "123")
	if err != nil {
		t.Fatalf("unexpected tokenize error: %v", err)
	}

	if tok.Brand != "visa" {
		t.Errorf("expected brand visa, got %s", tok.Brand)
	}

	if tok.Last4 != "4242" {
		t.Errorf("expected last4 4242, got %s", tok.Last4)
	}

	if tok.ExpMonth != 12 || tok.ExpYear != 2028 {
		t.Errorf("expected exp 12/2028, got %d/%d", tok.ExpMonth, tok.ExpYear)
	}

	// Detokenize
	fetched, err := v.Detokenize(ctx, tok.TokenID)
	if err != nil {
		t.Fatalf("unexpected detokenize error: %v", err)
	}

	if fetched.TokenID != tok.TokenID {
		t.Errorf("expected token ID %s, got %s", tok.TokenID, fetched.TokenID)
	}

	// GetByFingerprint
	byFp, err := v.GetByFingerprint(ctx, tok.Fingerprint)
	if err != nil {
		t.Fatalf("unexpected GetByFingerprint error: %v", err)
	}
	if byFp == nil || byFp.TokenID != tok.TokenID {
		t.Errorf("expected token matching fingerprint %s", tok.Fingerprint)
	}

	// Delete
	err = v.DeleteToken(ctx, tok.TokenID)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	_, err = v.Detokenize(ctx, tok.TokenID)
	if err == nil {
		t.Fatal("expected error detokenizing deleted token")
	}
}

func TestMockVault_CardBrands(t *testing.T) {
	v := vault.NewMockVault()
	ctx := context.Background()

	tests := []struct {
		card  string
		brand string
	}{
		{"4000000000000341", "visa"},
		{"5100000000000000", "mastercard"},
		{"370000000000000", "amex"},
		{"600000000000000", "unknown"},
	}

	for _, tt := range tests {
		tok, err := v.Tokenize(ctx, tt.card, "01", "2030", "999")
		if err != nil {
			t.Fatalf("failed to tokenize card %s: %v", tt.card, err)
		}
		if tok.Brand != tt.brand {
			t.Errorf("card %s: expected brand %s, got %s", tt.card, tt.brand, tok.Brand)
		}
	}
}
