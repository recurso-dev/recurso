package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

// Team management is owner/admin-only. requireManager runs before the auth
// service is touched, so these RBAC/validation paths need no database.
func newTeamHandlerNoDB() *TeamHandler {
	return NewTeamHandler(service.NewAuthService(nil, nil, nil, 0))
}

func TestInviteUser_NonManagerForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTeamHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/users/invite", `{"email":"x@y.com","role":"member"}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member") // not owner/admin
	h.InviteUser(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — inviting requires owner/admin", w.Code)
	}
}

func TestCreateUser_NonManagerForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTeamHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/users", `{"email":"x@y.com","role":"member","password":"secret12"}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member")
	h.CreateUser(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — creating a user requires owner/admin", w.Code)
	}
}

func TestInviteUser_ManagerBadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTeamHandlerNoDB()
	// Owner passes the RBAC gate; a malformed body then 400s before the service.
	c, w := jsonCtx(http.MethodPost, "/v1/users/invite", `{ not json `)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "owner")
	h.InviteUser(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a malformed invite body", w.Code)
	}
}
