package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/importer/revenuecat"
	"github.com/recurso-dev/recurso/internal/service"
)

// RevenueCatImportHandler serves the RevenueCat migration endpoints: a
// no-side-effect preview and an idempotent commit.
type RevenueCatImportHandler struct {
	svc *service.RevenueCatImportService
}

func NewRevenueCatImportHandler(svc *service.RevenueCatImportService) *RevenueCatImportHandler {
	return &RevenueCatImportHandler{svc: svc}
}

// Preview — POST /v1/import/revenuecat/preview
func (h *RevenueCatImportHandler) Preview(c *gin.Context) {
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

// Commit — POST /v1/import/revenuecat/commit
// Compare is the migration gate for RevenueCat — same contract as the Stripe
// and Chargebee gates: coverage, fidelity, continuity, zero writes.
//
// POST /v1/import/revenuecat/compare
func (h *RevenueCatImportHandler) Compare(c *gin.Context) {
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

func (h *RevenueCatImportHandler) Commit(c *gin.Context) {
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

func (h *RevenueCatImportHandler) readRequest(c *gin.Context) (uuid.UUID, *revenuecat.Export, bool) {
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "empty request body; upload a RevenueCat export")
		return uuid.Nil, nil, false
	}

	exp, err := revenuecat.Parse(body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return uuid.Nil, nil, false
	}
	return tenantID, exp, true
}
