package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// MeteringHandler exposes usage-based billing v1 (spec_usage_billing.md):
//
//	POST   /v1/billable-metrics                   create a metric
//	GET    /v1/billable-metrics                   list the tenant's metrics
//	GET    /v1/billable-metrics/:id               fetch one metric
//	PUT    /v1/billable-metrics/:id               update (code immutable)
//	DELETE /v1/billable-metrics/:id               delete (409 when charged)
//	PUT    /v1/plans/:id/charges                  replace a plan's charge set
//	GET    /v1/plans/:id/charges                  list a plan's charges
//	GET    /v1/subscriptions/:id/usage-amount     live pre-invoice preview
type MeteringHandler struct {
	svc *service.MeteringService
}

func NewMeteringHandler(svc *service.MeteringService) *MeteringHandler {
	return &MeteringHandler{svc: svc}
}

func meteringTenantCtx(c *gin.Context) (uuid.UUID, context.Context, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, nil, false
	}
	return tenantID, context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

// respondMeteringError maps service errors onto the canonical envelope.
func respondMeteringError(c *gin.Context, err error) {
	var valErr service.MeteringValidationError
	var ratingErr service.RatingError
	switch {
	case errors.Is(err, service.ErrMetricNotFound),
		errors.Is(err, service.ErrMeteringPlanNotFound),
		errors.Is(err, service.ErrUsageSubscriptionNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, service.ErrMetricCodeExists),
		errors.Is(err, service.ErrMetricInUse):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.As(err, &valErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
	case errors.As(err, &ratingErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, ratingErr.Error())
	default:
		respondInternalError(c, err)
	}
}

// CreateMetric handles POST /v1/billable-metrics.
func (h *MeteringHandler) CreateMetric(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	var in service.MetricInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	m, err := h.svc.CreateMetric(ctx, tenantID, in)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": m})
}

// ListMetrics handles GET /v1/billable-metrics.
func (h *MeteringHandler) ListMetrics(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	metrics, err := h.svc.ListMetrics(ctx, tenantID)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetMetric handles GET /v1/billable-metrics/:id.
func (h *MeteringHandler) GetMetric(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid metric id")
		return
	}
	m, err := h.svc.GetMetric(ctx, tenantID, id)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// UpdateMetric handles PUT /v1/billable-metrics/:id.
func (h *MeteringHandler) UpdateMetric(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid metric id")
		return
	}
	var in service.MetricInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	m, err := h.svc.UpdateMetric(ctx, tenantID, id, in)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// DeleteMetric handles DELETE /v1/billable-metrics/:id.
func (h *MeteringHandler) DeleteMetric(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid metric id")
		return
	}
	if err := h.svc.DeleteMetric(ctx, tenantID, id); err != nil {
		respondMeteringError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// SetPlanCharges handles PUT /v1/plans/:id/charges. The body is a JSON
// array of {metric_id, charge_model, amounts, hsn_code}; the plan's charge
// set is fully replaced (mirrors PUT /v1/plans/:id/entitlements).
func (h *MeteringHandler) SetPlanCharges(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}
	var inputs []service.ChargeInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	charges, err := h.svc.SetPlanCharges(ctx, tenantID, planID, inputs)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": charges})
}

// SimulateCharges handles POST /v1/plans/:id/simulate-charges — a read-only
// preview of what a proposed charge set would bill against sample usage, plus a
// balanced GL projection. Nothing is persisted.
func (h *MeteringHandler) SimulateCharges(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}
	var req service.SimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	sim, err := h.svc.SimulateCharges(ctx, tenantID, planID, req)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sim})
}

// GetPlanCharges handles GET /v1/plans/:id/charges.
func (h *MeteringHandler) GetPlanCharges(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}
	charges, err := h.svc.GetPlanCharges(ctx, tenantID, planID)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": charges})
}

// GetUsageAmount handles GET /v1/subscriptions/:id/usage-amount — the live
// preview of what the current period's usage would rate to if invoiced now.
func (h *MeteringHandler) GetUsageAmount(c *gin.Context) {
	tenantID, ctx, ok := meteringTenantCtx(c)
	if !ok {
		return
	}
	subID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription id")
		return
	}
	amount, err := h.svc.GetUsageAmount(ctx, tenantID, subID)
	if err != nil {
		respondMeteringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": amount})
}
