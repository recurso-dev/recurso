package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

// These exercise the dispute handler's request-validation paths, which all
// return before the service/repo is touched — so a service with a nil repo is
// sufficient and no database is required.
func newDisputeHandlerNoDB() *DisputeHandler {
	return NewDisputeHandler(service.NewDisputeService(nil))
}

func TestResolveDispute_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newDisputeHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/disputes/not-a-uuid/resolve", `{}`)
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.ResolveDispute(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an invalid dispute id", w.Code)
	}
}

func TestResolveDispute_InvalidOutcome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newDisputeHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/disputes/x/resolve", `{"outcome":"maybe"}`)
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	h.ResolveDispute(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an outcome outside accept/reject", w.Code)
	}
}

func TestListDisputes_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newDisputeHandlerNoDB()
	c, w := jsonCtx(http.MethodGet, "/v1/disputes?status=bogus", "")
	c.Set("tenant_id", uuid.New())
	h.ListDisputes(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an unknown status filter", w.Code)
	}
}

// A rejected outcome is accepted by validation (the 400 guard must NOT trip);
// it fails later only when the nil repo is reached, which is a 500, not a 400 —
// proving the validation layer let it through.
func TestResolveDispute_RejectOutcomeReachesService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newDisputeHandlerNoDB()
	c, w := jsonCtx(http.MethodPost, "/v1/disputes/x/resolve", `{"outcome":"reject"}`)
	c.Set("tenant_id", uuid.New())
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	// The nil repo will panic/error inside the service; recover so the test only
	// asserts that validation did NOT reject a valid outcome with a 400.
	defer func() { _ = recover() }()
	h.ResolveDispute(c)
	if w.Code == http.StatusBadRequest {
		t.Fatalf("a valid 'reject' outcome must not be rejected by validation (got 400)")
	}
}
