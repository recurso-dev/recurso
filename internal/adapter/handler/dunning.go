package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type DunningHandler struct {
	svc         *service.DunningAnalyticsService
	recoverySvc *service.DunningRecoveryService
}

func NewDunningHandler(svc *service.DunningAnalyticsService, recoverySvc *service.DunningRecoveryService) *DunningHandler {
	return &DunningHandler{svc: svc, recoverySvc: recoverySvc}
}

func (h *DunningHandler) GetOverview(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	overview, err := h.svc.GetOverview(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to fetch dunning overview")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": overview})
}

func (h *DunningHandler) GetWeights(c *gin.Context) {
	weights, err := h.svc.GetWeightsByContext(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to fetch dunning weights")
		return
	}
	if weights == nil {
		weights = []domain.DunningWeight{}
	}
	c.JSON(http.StatusOK, gin.H{"data": weights})
}

func (h *DunningHandler) GetHistory(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	history, err := h.svc.GetRecentHistory(c.Request.Context(), tenantID, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to fetch dunning history")
		return
	}
	if history == nil {
		history = []domain.DunningHistory{}
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}

// GetRecovered returns tenant-scoped recovered-revenue totals and a
// last-12-months monthly series.
func (h *DunningHandler) GetRecovered(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	summary, err := h.recoverySvc.GetRecoveredSummary(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to fetch recovered revenue")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}
