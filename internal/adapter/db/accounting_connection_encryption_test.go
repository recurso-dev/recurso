package db

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func testBox(t *testing.T) *secretbox.Box {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	b, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAccountingTokenSealOpenRoundTrip(t *testing.T) {
	r := &AccountingConnectionRepository{}
	r.SetVault(testBox(t))

	sealed, err := r.sealToken("ya29.secret-access-token")
	if err != nil {
		t.Fatalf("sealToken: %v", err)
	}
	if sealed == "" || sealed == "ya29.secret-access-token" {
		t.Fatalf("token not sealed: %q", sealed)
	}
	if got := r.openToken(sealed); got != "ya29.secret-access-token" {
		t.Fatalf("openToken round-trip: got %q", got)
	}
}

func TestAccountingOpenTokenFallsBackToPlaintext(t *testing.T) {
	r := &AccountingConnectionRepository{}
	r.SetVault(testBox(t))
	// A legacy plaintext token (not sealed with our key) must read through
	// unchanged rather than error — the migration-tolerant path.
	if got := r.openToken("legacy-plaintext-token"); got != "legacy-plaintext-token" {
		t.Fatalf("plaintext fallback: got %q", got)
	}
	if got := r.openToken(""); got != "" {
		t.Fatalf("empty token: got %q", got)
	}
}

func TestAccountingNoVaultIsPassthrough(t *testing.T) {
	r := &AccountingConnectionRepository{} // no vault
	sealed, err := r.sealToken("tok")
	if err != nil || sealed != "tok" {
		t.Fatalf("no-vault seal should passthrough: %q, %v", sealed, err)
	}
	if got := r.openToken("tok"); got != "tok" {
		t.Fatalf("no-vault open should passthrough: %q", got)
	}
}

func TestAccountingDecryptTokensBothFields(t *testing.T) {
	r := &AccountingConnectionRepository{}
	r.SetVault(testBox(t))
	access, _ := r.sealToken("access")
	refresh, _ := r.sealToken("refresh")
	c := &domain.AccountingConnection{AccessToken: access, RefreshToken: refresh}
	r.decryptTokens(c)
	if c.AccessToken != "access" || c.RefreshToken != "refresh" {
		t.Fatalf("decryptTokens: access=%q refresh=%q", c.AccessToken, c.RefreshToken)
	}
}
