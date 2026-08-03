package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/importer/chargebee"
	"github.com/recurso-dev/recurso/internal/service"
)

// ChargebeeImportHandler serves the Chargebee migration endpoints: a
// no-side-effect preview and an idempotent commit.
type ChargebeeImportHandler struct {
	svc *service.ChargebeeImportService
}

func NewChargebeeImportHandler(svc *service.ChargebeeImportService) *ChargebeeImportHandler {
	return &ChargebeeImportHandler{svc: svc}
}

// Preview parses an uploaded Chargebee export and returns a dry-run plan — zero
// side effects. POST /v1/import/chargebee/preview
func (h *ChargebeeImportHandler) Preview(c *gin.Context) {
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

// Commit imports the uploaded Chargebee export (customers, plans, subscriptions)
// idempotently. Per-object failures are reported, not fatal.
// POST /v1/import/chargebee/commit
// Compare is the migration gate for Chargebee — same contract as the Stripe
// one: coverage, fidelity, billing continuity, zero writes.
//
// POST /v1/import/chargebee/compare
func (h *ChargebeeImportHandler) Compare(c *gin.Context) {
	tenantID, exp, ok := h.readRequest(c)
	if !ok {
		return
	}
	report, err := h.svc.Compare(c.Request.Context(), tenantID, exp)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *ChargebeeImportHandler) Commit(c *gin.Context) {
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

func (h *ChargebeeImportHandler) readRequest(c *gin.Context) (uuid.UUID, *chargebee.Export, bool) {
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "empty request body; upload a Chargebee export")
		return uuid.Nil, nil, false
	}

	exp, err := chargebee.Parse(body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return uuid.Nil, nil, false
	}
	return tenantID, exp, true
}
