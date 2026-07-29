package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

// Credit-note handler validation paths (bad id → 400, non-admin → 403) both
// return before the service is used, so a service with nil deps is enough and
// no database is required.
func newCreditNoteHandlerNoDB() *CreditNoteHandler {
	return NewCreditNoteHandler(service.NewCreditNoteService(nil, nil, nil, nil))
}

func TestVoidCreditNote_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newCreditNoteHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/credit-notes/nope/void", "")
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "owner")
	c.Params = gin.Params{{Key: "id", Value: "nope"}}
	h.VoidCreditNote(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an invalid credit note id", w.Code)
	}
}

func TestVoidCreditNote_NonAdminForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newCreditNoteHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/credit-notes/x/void", "")
	c.Set("tenant_id", uuid.New())
	c.Set("user_id", uuid.New())
	c.Set("user_role", "member") // not admin/owner
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	h.VoidCreditNote(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — only admins/owners may void a credit note", w.Code)
	}
}

func TestApproveCreditNote_NonAdminForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newCreditNoteHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/credit-notes/x/approve", "")
	c.Set("tenant_id", uuid.New())
	c.Set("user_id", uuid.New())
	c.Set("user_role", "member")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	h.ApproveCreditNote(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — only admins/owners may approve a credit note", w.Code)
	}
}
