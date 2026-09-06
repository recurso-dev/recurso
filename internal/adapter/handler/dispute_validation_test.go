package handler

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port/porttest"
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

// overTotalDisputeRepo hands back one open dispute; every other repository
// method panics by name (porttest), which proves the over-total check fires
// before Close is ever reached.
type overTotalDisputeRepo struct {
	porttest.UnimplementedDisputeRepository
	dispute *domain.InvoiceDispute
}

func (r *overTotalDisputeRepo) GetByID(context.Context, uuid.UUID) (*domain.InvoiceDispute, error) {
	return r.dispute, nil
}

type overTotalInvoiceReader struct{ invoice *domain.Invoice }

func (r overTotalInvoiceReader) GetByIDPublic(context.Context, uuid.UUID) (*domain.Invoice, error) {
	return r.invoice, nil
}

// A credit_amount above the invoice total is the caller's mistake, so it must
// surface as 400 validation_failed rather than the 500 it used to be.
func TestResolveDispute_OverTotalCreditIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	invID := uuid.New()
	dispID := uuid.New()
	svc := service.NewDisputeService(&overTotalDisputeRepo{dispute: &domain.InvoiceDispute{
		ID: dispID, TenantID: tenant, InvoiceID: invID, Status: domain.DisputeStatusOpen,
	}})
	svc.SetCreditIssuer(&service.CreditNoteService{}, overTotalInvoiceReader{invoice: &domain.Invoice{
		ID: invID, TenantID: tenant, Total: 5000, AmountDue: 5000,
	}})
	h := NewDisputeHandler(svc)

	c, w := jsonCtx(http.MethodPost, "/v1/disputes/x/resolve", `{"outcome":"accept","issue_credit":true,"credit_amount":5001}`)
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "id", Value: dispID.String()}}
	h.ResolveDispute(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a credit above the invoice total; body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "validation_failed") {
		t.Fatalf("body = %s, want the validation_failed code", w.Body.String())
	}
}
