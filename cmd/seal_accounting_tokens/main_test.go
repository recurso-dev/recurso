package main

import (
	"crypto/rand"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
)

func TestTokenSealingLogic(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	box, err := secretbox.New(key)
	if err != nil {
		t.Fatalf("unexpected error creating secretbox: %v", err)
	}

	plainToken := "access_tok_plaintext_12345"

	// Verify initially unencrypted
	_, err = box.Open(plainToken)
	if err == nil {
		t.Fatal("expected plaintext token to fail Open")
	}

	// Seal token
	sealed, err := box.Seal(plainToken)
	if err != nil {
		t.Fatalf("unexpected error sealing token: %v", err)
	}

	// Open sealed token
	unsealed, err := box.Open(sealed)
	if err != nil {
		t.Fatalf("unexpected error opening sealed token: %v", err)
	}

	if unsealed != plainToken {
		t.Errorf("expected %s, got %s", plainToken, unsealed)
	}
}
