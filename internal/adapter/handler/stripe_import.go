package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
	"github.com/recurso-dev/recurso/internal/service"
)

// maxImportBytes caps an uploaded Stripe export so a huge (or hostile) body
// can't exhaust memory. 25 MB comfortably holds tens of thousands of objects.
const maxImportBytes = 25 << 20

// StripeImportHandler serves the Stripe migration endpoints: a no-side-effect
// preview (dry run) and an idempotent commit. Both take the Stripe export JSON
// as the request body.
type StripeImportHandler struct {
	svc *service.StripeImportService
}

func NewStripeImportHandler(svc *service.StripeImportService) *StripeImportHandler {
	return &StripeImportHandler{svc: svc}
}

// Preview parses an uploaded Stripe export and returns a dry-run plan of exactly
// what a commit would create, link, skip, or refuse — with zero side effects.
//
// POST /v1/import/stripe/preview
func (h *StripeImportHandler) Preview(c *gin.Context) {
	tenantID, exp, ok := h.readRequest(c)
	if !ok {
		return
	}
	plan, err := h.svc.Preview(c.Request.Context(), tenantID, exp)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

// Commit imports the uploaded Stripe export — creating customers and plans and
// recording an idempotency ref for each. Re-running is safe (already-imported
// ids and existing emails/plan-codes are skipped). Per-object failures are
// reported in the response, not fatal.
//
// POST /v1/import/stripe/commit
func (h *StripeImportHandler) Commit(c *gin.Context) {
	tenantID, exp, ok := h.readRequest(c)
	if !ok {
		return
	}
	result, err := h.svc.Commit(c.Request.Context(), tenantID, exp)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// readRequest resolves the tenant and parses the Stripe export body, writing the
// error response itself and returning ok=false on any failure.
func (h *StripeImportHandler) readRequest(c *gin.Context) (uuid.UUID, *stripeimport.Export, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "missing tenant")
		return uuid.Nil, nil, false
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxImportBytes+1))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "could not read request body")
		return uuid.Nil, nil, false
	}
	if len(body) > maxImportBytes {
		respondError(c, http.StatusRequestEntityTooLarge, codeValidationFailed, "export exceeds the 25 MB limit")
		return uuid.Nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "empty request body; upload a Stripe export")
		return uuid.Nil, nil, false
	}

	exp, err := stripeimport.Parse(body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return uuid.Nil, nil, false
	}
	return tenantID, exp, true
}
