package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Organization id-parse validation returns 400 before the service is used.
func TestGetOrganization_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOrganizationHandler(nil)
	c, w := jsonCtx(http.MethodGet, "/v1/organizations/not-a-uuid", "")
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "owner")
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.GetOrganization(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an invalid organization id", w.Code)
	}
}

func TestAddTenant_InvalidOrgID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOrganizationHandler(nil)
	c, w := jsonCtx(http.MethodPost, "/v1/organizations/bad/tenants", `{"tenant_id":"x"}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "owner")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	h.AddTenant(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an invalid organization id", w.Code)
	}
}
