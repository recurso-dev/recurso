package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
	Currency      string `json:"currency" binding:"required,currency"`
	HSNCode       string `json:"hsn_code"`
}

func (h *CatalogHandler) CreatePlan(c *gin.Context) {
	var req createPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	// `required` already rejects a zero amount; also reject a negative one so a
	// plan can never bill a negative recurring charge.
	if req.Amount < 0 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "amount must be positive")
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
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
		HSNCode:       req.HSNCode,
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	plan, err := h.service.CreatePlan(ctx, input)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": plan})
}

func (h *CatalogHandler) GetPlan(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	plan, err := h.service.GetPlan(ctx, tenantID, planID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if plan == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "plan not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plan})
}

// updatePlanRequest is a partial update: nil fields are left unchanged, so the
// same endpoint edits plan metadata or archives it (active=false).
type updatePlanRequest struct {
	Name          *string `json:"name" binding:"omitempty,min=1"`
	HSNCode       *string `json:"hsn_code"`
	IntervalUnit  *string `json:"interval_unit" binding:"omitempty,oneof=day week month year"`
	IntervalCount *int    `json:"interval_count" binding:"omitempty,min=1"`
	Active        *bool   `json:"active"`
}

func (h *CatalogHandler) UpdatePlan(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}

	var req updatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	plan, err := h.service.UpdatePlan(ctx, service.UpdatePlanInput{
		TenantID:      tenantID,
		PlanID:        planID,
		Name:          req.Name,
		HSNCode:       req.HSNCode,
		IntervalUnit:  req.IntervalUnit,
		IntervalCount: req.IntervalCount,
		Active:        req.Active,
	})
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if plan == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "plan not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *CatalogHandler) ListPlans(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	// Parse query params
	search := c.Query("q")

	limit, offset := parsePageLimit(c)

	filter := domain.PlanFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	plans, err := h.service.ListPlans(ctx, tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if plans == nil {
		plans = []*domain.Plan{}
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}
