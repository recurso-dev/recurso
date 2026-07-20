package secretbox

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
)

func newTestBox(t *testing.T) *Box {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	b, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

func TestSealOpenRoundTrip(t *testing.T) {
	b := newTestBox(t)
	for _, plain := range []string{"sk_live_secret", "rzp_test_123", "a", strings.Repeat("x", 4096)} {
		sealed, err := b.Seal(plain)
		if err != nil {
			t.Fatalf("Seal(%q): %v", plain, err)
		}
		if sealed == plain {
			t.Fatalf("ciphertext equals plaintext for %q", plain)
		}
		got, err := b.Open(sealed)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if got != plain {
			t.Fatalf("round trip: got %q want %q", got, plain)
		}
	}
}

func TestEmptyRoundTrips(t *testing.T) {
	b := newTestBox(t)
	sealed, err := b.Seal("")
	if err != nil || sealed != "" {
		t.Fatalf("Seal(\"\") = %q, %v; want \"\", nil", sealed, err)
	}
	got, err := b.Open("")
	if err != nil || got != "" {
		t.Fatalf("Open(\"\") = %q, %v; want \"\", nil", got, err)
	}
}

func TestNonceIsRandom(t *testing.T) {
	b := newTestBox(t)
	a, _ := b.Seal("same-plaintext")
	c, _ := b.Seal("same-plaintext")
	if a == c {
		t.Fatal("two seals of the same plaintext produced identical ciphertext (nonce reuse)")
	}
}

func TestWrongKeyFailsToOpen(t *testing.T) {
	b1 := newTestBox(t)
	b2 := newTestBox(t)
	sealed, _ := b1.Seal("secret")
	if _, err := b2.Open(sealed); err == nil {
		t.Fatal("Open with a different key must fail")
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	b := newTestBox(t)
	sealed, _ := b.Seal("secret")
	raw, _ := base64.StdEncoding.DecodeString(sealed)
	raw[len(raw)-1] ^= 0xFF // flip the last tag byte
	tampered := base64.StdEncoding.EncodeToString(raw)
	if _, err := b.Open(tampered); err == nil {
		t.Fatal("Open of a tampered ciphertext must fail (GCM auth)")
	}
}

func TestNewRejectsBadKeyLength(t *testing.T) {
	if _, err := New(make([]byte, 16)); err == nil {
		t.Fatal("New must reject a 16-byte key")
	}
}

func TestNewFromEnvValue(t *testing.T) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"std base64": base64.StdEncoding.EncodeToString(key),
		"raw base64": base64.RawStdEncoding.EncodeToString(key),
		"hex":        hex.EncodeToString(key),
	}
	for name, encoded := range cases {
		if _, err := NewFromEnvValue(encoded); err != nil {
			t.Fatalf("%s: NewFromEnvValue: %v", name, err)
		}
	}

	if _, err := NewFromEnvValue(""); !errors.Is(err, ErrNoKey) {
		t.Fatalf("empty value: got %v, want ErrNoKey", err)
	}
	if _, err := NewFromEnvValue("too-short"); err == nil {
		t.Fatal("a non-32-byte key must be rejected")
	}
}

func TestOpenRejectsGarbage(t *testing.T) {
	b := newTestBox(t)
	if _, err := b.Open("!!!not base64!!!"); err == nil {
		t.Fatal("Open must reject non-base64 input")
	}
	if _, err := b.Open("YWJj"); err == nil { // "abc" — shorter than a nonce
		t.Fatal("Open must reject too-short ciphertext")
	}
}
