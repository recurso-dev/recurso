package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

type RevRecHandler struct {
	revrecService *service.RevRecService
}

func NewRevRecHandler(revrecService *service.RevRecService) *RevRecHandler {
	return &RevRecHandler{revrecService: revrecService}
}

// GetWaterfall returns the caller's tenant recognized-plus-scheduled revenue
// curve, month by month, with totals. Read-only.
func (h *RevRecHandler) GetWaterfall(c *gin.Context) {
	tid, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	waterfall, err := h.revrecService.GetWaterfall(c.Request.Context(), tid)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": waterfall})
}

func (h *RevRecHandler) GetReport(c *gin.Context) {
	tid, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	monthStr := c.Query("month")
	yearStr := c.Query("year")

	if monthStr == "" || yearStr == "" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "month and year query parameters are required")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid month")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid year")
		return
	}

	report, err := h.revrecService.GetReport(c.Request.Context(), tid, month, year)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}
