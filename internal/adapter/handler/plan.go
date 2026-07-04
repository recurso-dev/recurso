package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

type CatalogHandler struct {
	service *service.CatalogService
}

func NewCatalogHandler(s *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: s}
}

type createPlanRequest struct {
	Name          string `json:"name" binding:"required"`
	Code          string `json:"code" binding:"required"`
	IntervalUnit  string `json:"interval_unit" binding:"required,oneof=day week month year"`
	IntervalCount int    `json:"interval_count" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required"`
	Currency      string `json:"currency" binding:"required,len=3"`
}

func (h *CatalogHandler) CreatePlan(c *gin.Context) {
	var req createPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	input := service.CreatePlanInput{
		TenantID:      tenantID,
		Name:          req.Name,
		Code:          req.Code,
		IntervalUnit:  req.IntervalUnit,
		IntervalCount: req.IntervalCount,
		Amount:        req.Amount,
		Currency:      req.Currency,
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	plan, err := h.service.CreatePlan(ctx, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, plan)
}

func (h *CatalogHandler) ListPlans(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	// Parse query params
	search := c.Query("q")

	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	offset := 0
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			offset = (v - 1) * limit
		}
	}

	filter := domain.PlanFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	plans, err := h.service.ListPlans(ctx, tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if plans == nil {
		plans = []*domain.Plan{}
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}
