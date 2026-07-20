package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

type ClosePackHandler struct {
	service *service.ClosePackService
}

func NewClosePackHandler(service *service.ClosePackService) *ClosePackHandler {
	return &ClosePackHandler{service: service}
}

// GetClosePack returns the caller's month-end close pack for ?month=&year=:
// trial balance, reconciliation status, the deferred-revenue rollforward, a
// pointer to the GL export, and a ready-to-close verdict with blockers.
// Read-only.
func (h *ClosePackHandler) GetClosePack(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid or missing month (1-12)")
		return
	}
	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2000 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid or missing year")
		return
	}

	pack, err := h.service.Generate(c.Request.Context(), tenantID.(uuid.UUID), month, year)
	if err != nil {
		log.Printf("close pack error: %v", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to build close pack")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pack})
}
