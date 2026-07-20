package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/secretbox"
	"github.com/recurso-dev/recurso/internal/service"
)

// newHandler builds a real GatewayConnectionService over a nil-DB repo — enough
// to exercise the validation and vault-availability branches that never touch
// the database.
func newHandler(t *testing.T, withVault bool) *GatewayConnectionHandler {
	t.Helper()
	var box *secretbox.Box
	if withVault {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i + 1)
		}
		var err error
		box, err = secretbox.New(key)
		if err != nil {
			t.Fatal(err)
		}
	}
	svc := service.NewGatewayConnectionService(db.NewGatewayConnectionRepository(nil), box)
	return NewGatewayConnectionHandler(svc)
}

func postConnect(h *GatewayConnectionHandler, body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", uuid.New())
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/gateway-connections", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Connect(c)
	return w
}

func TestConnectHandler_VaultUnavailable(t *testing.T) {
	h := newHandler(t, false)
	w := postConnect(h, map[string]string{"provider": "stripe", "secret_key": "sk_test_x"})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no vault: got %d want 503", w.Code)
	}
}

func TestConnectHandler_ValidationError(t *testing.T) {
	h := newHandler(t, true)
	// Unknown provider -> 400 (validation) before any DB access.
	w := postConnect(h, map[string]string{"provider": "paypal", "secret_key": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad provider: got %d want 400", w.Code)
	}
}

func TestConnectHandler_MissingRequiredField(t *testing.T) {
	h := newHandler(t, true)
	// Missing secret_key -> binding 400.
	w := postConnect(h, map[string]string{"provider": "stripe"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing secret: got %d want 400", w.Code)
	}
}

func TestConnectHandler_MissingTenant(t *testing.T) {
	h := newHandler(t, true)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No tenant_id set.
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/gateway-connections", bytes.NewReader([]byte(`{}`)))
	h.Connect(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no tenant: got %d want 401", w.Code)
	}
}
