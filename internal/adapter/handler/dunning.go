package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

type DunningHandler struct {
	svc *service.DunningAnalyticsService
}

func NewDunningHandler(svc *service.DunningAnalyticsService) *DunningHandler {
	return &DunningHandler{svc: svc}
}

func (h *DunningHandler) GetOverview(c *gin.Context) {
	overview, err := h.svc.GetOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dunning overview"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *DunningHandler) GetWeights(c *gin.Context) {
	weights, err := h.svc.GetWeightsByContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dunning weights"})
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

	history, err := h.svc.GetRecentHistory(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dunning history"})
		return
	}
	if history == nil {
		history = []domain.DunningHistory{}
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}
