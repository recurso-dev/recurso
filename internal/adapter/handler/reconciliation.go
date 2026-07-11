package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

// ReconciliationHandler exposes on-demand ledger reconciliation.
type ReconciliationHandler struct {
	service *service.ReconciliationService
}

func NewReconciliationHandler(service *service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{service: service}
}

// RunReconciliation runs a ledger-vs-billing reconciliation for the caller's
// tenant and returns the computed report. Nothing is persisted.
func (h *ReconciliationHandler) RunReconciliation(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	report, err := h.service.Run(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to run reconciliation")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}
