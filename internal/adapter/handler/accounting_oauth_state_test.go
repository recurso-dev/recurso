package handler

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestAccountingOAuthState_RoundTripAndForgery proves the ENG-166 M1 fix: the
// OAuth `state` is bound to the handler's secret. A handler with a different
// secret (what an attacker gets once the old hardcoded fallback key is public
// in the open-source repo) cannot forge a state the real handler will accept.
func TestAccountingOAuthState_RoundTripAndForgery(t *testing.T) {
	real := &AccountingHandler{oauthStateSecret: []byte("real-server-secret-32-bytes-long!")}
	attacker := &AccountingHandler{oauthStateSecret: []byte("recurso-oauth-state-fallback-key")}

	tenantID := uuid.New()

	// Round-trip under the real secret recovers the tenant.
	state := real.generateOAuthState(tenantID, "quickbooks")
	got, err := real.verifyOAuthState(state)
	if err != nil {
		t.Fatalf("verify own state: %v", err)
	}
	if got != tenantID {
		t.Fatalf("round-trip tenant = %s, want %s", got, tenantID)
	}

	// A state forged with a different (e.g. leaked/hardcoded) secret is rejected.
	forged := attacker.generateOAuthState(tenantID, "quickbooks")
	if _, err := real.verifyOAuthState(forged); err == nil {
		t.Fatal("forged state accepted: HMAC secret is not actually enforced")
	}

	// A tampered signature is rejected.
	tampered := state[:len(state)-1] + flipLast(state)
	if _, err := real.verifyOAuthState(tampered); err == nil {
		t.Fatal("tampered state accepted")
	}

	// A malformed state is rejected.
	if _, err := real.verifyOAuthState("not-a-valid-state"); err == nil {
		t.Fatal("malformed state accepted")
	}
}

func flipLast(s string) string {
	last := s[len(s)-1]
	if last == '0' {
		return "1"
	}
	if strings.ContainsRune("123456789abcdef", rune(last)) {
		return "0"
	}
	return "0"
}
