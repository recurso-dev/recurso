package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/service"
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	report, err := h.service.Run(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run reconciliation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}
