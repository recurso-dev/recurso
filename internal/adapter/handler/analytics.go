package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req struct {
		Question string `json:"question" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	data, sqlQuery, err := h.genaiSvc.Ask(c.Request.Context(), tenantID, req.Question)
	if err != nil {
		if errors.Is(err, service.ErrGenAINotConfigured) {
			respondError(c, http.StatusServiceUnavailable, codeInternalError, err.Error())
			return
		}
		respondInternalError(c, err)
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	// Inject TenantID into context for Service/Repo
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	mrr, err := h.svc.GetMRR(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to calculate MRR")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mrr})
}

// GetMRRWaterfall returns the MRR movement breakdown between two dates
// (?start=YYYY-MM-DD&end=YYYY-MM-DD; default = the trailing month).
func (h *AnalyticsHandler) GetMRRWaterfall(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	end := time.Now()
	start := end.AddDate(0, -1, 0) // default: trailing month
	if v := c.Query("end"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid end date (want YYYY-MM-DD)")
			return
		}
		end = t
	}
	if v := c.Query("start"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid start date (want YYYY-MM-DD)")
			return
		}
		start = t
	}
	if !start.Before(end) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "start must be before end")
		return
	}

	wf, err := h.svc.GetMRRWaterfall(ctx, tenantID, start, end)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to compute MRR waterfall")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": wf})
}

// GetInvoiceAging returns outstanding receivables bucketed by days past due.
func (h *AnalyticsHandler) GetInvoiceAging(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	report, err := h.svc.GetInvoiceAging(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to compute invoice aging")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetUnitEconomics returns ARPA, ARPU and LTV for the tenant.
func (h *AnalyticsHandler) GetUnitEconomics(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	ue, err := h.svc.GetUnitEconomics(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to compute unit economics")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ue})
}

// GetRevenueByPlan returns MRR broken down by plan.
func (h *AnalyticsHandler) GetRevenueByPlan(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	report, err := h.svc.GetRevenueByPlan(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to compute revenue by plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetRevenueByGeography returns MRR broken down by customer country.
func (h *AnalyticsHandler) GetRevenueByGeography(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	report, err := h.svc.GetRevenueByGeography(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to compute revenue by geography")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

func (h *AnalyticsHandler) GetUsageStats(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	stats, err := h.svc.GetUsageStats(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to fetch usage stats")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
