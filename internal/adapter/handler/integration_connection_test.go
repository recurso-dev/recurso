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

func newIntegrationHandler(t *testing.T, withVault bool) *IntegrationConnectionHandler {
	t.Helper()
	var box *secretbox.Box
	if withVault {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i + 7)
		}
		var err error
		box, err = secretbox.New(key)
		if err != nil {
			t.Fatal(err)
		}
	}
	svc := service.NewIntegrationConnectionService(db.NewIntegrationConnectionRepository(nil), box)
	return NewIntegrationConnectionHandler(svc)
}

func postIntegration(h *IntegrationConnectionHandler, body any, withTenant bool) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if withTenant {
		c.Set("tenant_id", uuid.New())
	}
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/integration-connections", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Connect(c)
	return w
}

func TestIntegrationConnect_VaultUnavailable(t *testing.T) {
	h := newIntegrationHandler(t, false)
	w := postIntegration(h, map[string]any{"category": "tax", "provider": "taxjar", "config": map[string]string{"api_key": "x"}}, true)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no vault: got %d want 503", w.Code)
	}
}

func TestIntegrationConnect_Validation(t *testing.T) {
	h := newIntegrationHandler(t, true)
	// Unknown provider -> 400 before DB.
	w := postIntegration(h, map[string]any{"category": "tax", "provider": "vertex", "config": map[string]string{"api_key": "x"}}, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad provider: got %d want 400", w.Code)
	}
}

func TestIntegrationConnect_MissingBody(t *testing.T) {
	h := newIntegrationHandler(t, true)
	w := postIntegration(h, map[string]any{"category": "tax"}, true) // no provider/config
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing fields: got %d want 400", w.Code)
	}
}

func TestIntegrationConnect_MissingTenant(t *testing.T) {
	h := newIntegrationHandler(t, true)
	w := postIntegration(h, map[string]any{"category": "tax", "provider": "taxjar", "config": map[string]string{"api_key": "x"}}, false)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no tenant: got %d want 401", w.Code)
	}
}
