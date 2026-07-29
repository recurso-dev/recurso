package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

func newTenantHandlerNoDB() *TenantHandler {
	return NewTenantHandler(&service.TenantService{})
}

// API-key creation is owner/admin-only; requireManager runs before the service.
func TestCreateKey_NonManagerForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTenantHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/developer/keys", `{}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member") // not owner/admin
	h.CreateKey(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — creating an API key requires owner/admin", w.Code)
	}
}

// Public tenant registration rejects a malformed email at the binding layer.
func TestRegister_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTenantHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/register", `{"name":"Acme","email":"not-an-email"}`)
	h.Register(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a malformed registration email", w.Code)
	}
}
