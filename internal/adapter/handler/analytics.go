package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

type AnalyticsHandler struct {
	svc      *service.AnalyticsService
	genaiSvc *service.GenAIService
}

func NewAnalyticsHandler(svc *service.AnalyticsService, genaiSvc *service.GenAIService) *AnalyticsHandler {
	return &AnalyticsHandler{
		svc:      svc,
		genaiSvc: genaiSvc,
	}
}

func (h *AnalyticsHandler) Ask(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	var req struct {
		Question string `json:"question" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, sqlQuery, err := h.genaiSvc.Ask(c.Request.Context(), tenantID, req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "query": sqlQuery})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  data,
		"query": sqlQuery,
	})
}

func (h *AnalyticsHandler) GetMRR(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	// Inject TenantID into context for Service/Repo
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	mrr, err := h.svc.GetMRR(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate MRR"})
		return
	}
	c.JSON(http.StatusOK, mrr)
}

func (h *AnalyticsHandler) GetUsageStats(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	stats, err := h.svc.GetUsageStats(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch usage stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
