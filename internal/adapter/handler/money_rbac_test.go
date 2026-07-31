package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestMoneyMutations_RequireManagerRole proves the money-sensitive mutations
// (wallet close = refund residue out; offline payment = fabricate settlement)
// are owner/admin-only. A `member` is forbidden; a manager passes the gate.
// The 403 is written before any service call, so nil services are fine.
func TestMoneyMutations_RequireManagerRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Wallet close by a member → 403.
	wh := NewWalletHandler(nil)
	c, w := jsonCtx(http.MethodPost, "/v1/wallets/x/close", `{}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member")
	wh.Close(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("wallet close as member: status = %d, want 403", w.Code)
	}

	// Offline payment by a member → 403.
	oh := NewOfflinePaymentHandler(nil)
	c2, w2 := jsonCtx(http.MethodPost, "/v1/payments/offline", `{}`)
	c2.Set("tenant_id", uuid.New())
	c2.Set("user_role", "member")
	oh.RecordOfflinePayment(c2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("offline payment as member: status = %d, want 403", w2.Code)
	}

	// An admin passes the manager gate (the handler then fails body validation,
	// i.e. NOT 403 — proving the gate let the manager through).
	c3, w3 := jsonCtx(http.MethodPost, "/v1/payments/offline", `{}`)
	c3.Set("tenant_id", uuid.New())
	c3.Set("user_role", "admin")
	NewOfflinePaymentHandler(nil).RecordOfflinePayment(c3)
	if w3.Code == http.StatusForbidden {
		t.Errorf("offline payment as admin: got 403, want the manager gate to pass")
	}
}
