package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SetRegistrations gates writes to owner/admin via requireManager, which runs
// before the repo is touched — so the 403 path needs no database.
func TestSetRegistrations_NonManagerForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaxNexusHandler(nil)
	c, w := jsonCtx(http.MethodPut, "/v1/settings/nexus/registrations", `{"registrations":[]}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member") // not owner/admin
	h.SetRegistrations(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — nexus registrations are owner/admin-only", w.Code)
	}
}
