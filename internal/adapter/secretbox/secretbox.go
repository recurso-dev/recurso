// Package secretbox is the at-rest encryption used for tenant-supplied
// third-party credentials (payment-gateway API keys, and later tax/CRM/storage
// keys). Unlike the accounting-connection tokens — which are stored plaintext
// (a known gap) — anything routed through here is sealed with AES-256-GCM under
// a single instance master key (GATEWAY_ENCRYPTION_KEY).
//
// Ciphertext layout (base64 std-encoded): nonce (12 bytes) || GCM ciphertext+tag.
// The key is 32 raw bytes provided as base64 or hex; a wrong-length key fails at
// construction, never silently at charge time.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrNoKey is returned by NewFromEnvValue when the configured key is empty —
// callers use it to decide whether BYO credential storage is available.
var ErrNoKey = errors.New("secretbox: no encryption key configured")

// Box seals and opens secrets under one AES-256-GCM key.
type Box struct {
	aead cipher.AEAD
}

// New builds a Box from a 32-byte key.
func New(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secretbox: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// NewFromEnvValue builds a Box from a base64- or hex-encoded 32-byte key string
// (e.g. the value of GATEWAY_ENCRYPTION_KEY). An empty string returns ErrNoKey
// so the caller can degrade to env-only gateways instead of crashing.
func NewFromEnvValue(v string) (*Box, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, ErrNoKey
	}
	key, err := decodeKey(v)
	if err != nil {
		return nil, err
	}
	return New(key)
}

// decodeKey accepts standard base64, raw-URL base64, or hex; it must decode to
// exactly 32 bytes.
func decodeKey(v string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := hex.DecodeString(v); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, errors.New("secretbox: key must decode (base64 or hex) to exactly 32 bytes")
}

// Seal encrypts plaintext and returns base64(nonce || ciphertext). Empty
// plaintext seals to an empty string so optional fields round-trip cleanly.
func (b *Box) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Open reverses Seal. An empty string opens to empty (mirrors Seal).
func (b *Box) Open(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("secretbox: bad base64: %w", err)
	}
	ns := b.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("secretbox: ciphertext too short")
	}
	nonce, body := raw[:ns], raw[ns:]
	plain, err := b.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("secretbox: open failed (wrong key or tampered): %w", err)
	}
	return string(plain), nil
}
