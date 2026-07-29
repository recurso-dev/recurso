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

// ChargebeeImportHandler serves the Chargebee migration preview (dry run). It
// mirrors the Stripe importer; commit is a follow-up increment.
type ChargebeeImportHandler struct {
	svc *service.ChargebeeImportService
}

func NewChargebeeImportHandler(svc *service.ChargebeeImportService) *ChargebeeImportHandler {
	return &ChargebeeImportHandler{svc: svc}
}

// Preview parses an uploaded Chargebee export and returns a dry-run plan — with
// zero side effects.
//
// POST /v1/import/chargebee/preview
func (h *ChargebeeImportHandler) Preview(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "missing tenant")
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxImportBytes+1))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "could not read request body")
		return
	}
	if len(body) > maxImportBytes {
		respondError(c, http.StatusRequestEntityTooLarge, codeValidationFailed, "export exceeds the 25 MB limit")
		return
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "empty request body; upload a Chargebee export")
		return
	}

	exp, err := chargebee.Parse(body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	plan, err := h.svc.Preview(c.Request.Context(), tenantID, exp)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}
