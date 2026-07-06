package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

// EntitlementHandler exposes the Entitlement Engine v1 endpoints:
//
//	PUT /v1/plans/:id/entitlements        replace a plan's entitlement set
//	GET /v1/plans/:id/entitlements        list a plan's entitlement set
//	GET /v1/customers/:id/entitlements    customer's effective entitlements
//	GET /v1/entitlements/check            fast single-feature check
type EntitlementHandler struct {
	service *service.EntitlementService
}

func NewEntitlementHandler(s *service.EntitlementService) *EntitlementHandler {
	return &EntitlementHandler{service: s}
}

// SetPlanEntitlements handles PUT /v1/plans/:id/entitlements. The body is
// a JSON array of {feature_key, kind, bool_value|limit_value}; the plan's
// stored set is fully replaced (absent keys are removed).
func (h *EntitlementHandler) SetPlanEntitlements(c *gin.Context) {
	tenantID, ctx, ok := entitlementTenantCtx(c)
	if !ok {
		return
	}

	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}

	var inputs []service.EntitlementInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ents, err := h.service.SetPlanEntitlements(ctx, tenantID, planID, inputs)
	if err != nil {
		respondEntitlementError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ents})
}

// GetPlanEntitlements handles GET /v1/plans/:id/entitlements.
func (h *EntitlementHandler) GetPlanEntitlements(c *gin.Context) {
	tenantID, ctx, ok := entitlementTenantCtx(c)
	if !ok {
		return
	}

	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan id")
		return
	}

	ents, err := h.service.GetPlanEntitlements(ctx, tenantID, planID)
	if err != nil {
		respondEntitlementError(c, err)
		return
	}
	if ents == nil {
		ents = []domain.Entitlement{}
	}
	c.JSON(http.StatusOK, gin.H{"data": ents})
}

// GetCustomerEntitlements handles GET /v1/customers/:id/entitlements and
// returns the customer's effective set (union over active/trialing
// subscriptions' plans) with the contributing plan ids per feature.
func (h *EntitlementHandler) GetCustomerEntitlements(c *gin.Context) {
	tenantID, ctx, ok := entitlementTenantCtx(c)
	if !ok {
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}

	effective, err := h.service.GetCustomerEntitlements(ctx, tenantID, customerID)
	if err != nil {
		respondEntitlementError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": effective})
}

// CheckEntitlement handles GET /v1/entitlements/check?customer_id=&feature=
// — the hot path. Absent grants answer granted=false with a null limit.
func (h *EntitlementHandler) CheckEntitlement(c *gin.Context) {
	tenantID, ctx, ok := entitlementTenantCtx(c)
	if !ok {
		return
	}

	customerID, err := uuid.Parse(c.Query("customer_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "customer_id must be a valid uuid")
		return
	}

	check, err := h.service.CheckFeature(ctx, tenantID, customerID, c.Query("feature"))
	if err != nil {
		respondEntitlementError(c, err)
		return
	}
	c.JSON(http.StatusOK, check)
}

// entitlementTenantCtx extracts the authenticated tenant and returns a
// request context carrying it for tenant-scoped repositories.
func entitlementTenantCtx(c *gin.Context) (uuid.UUID, context.Context, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, nil, false
	}
	return tenantID, context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

// respondEntitlementError maps service errors onto the canonical envelope.
func respondEntitlementError(c *gin.Context, err error) {
	var valErr service.EntitlementValidationError
	switch {
	case errors.Is(err, service.ErrEntitlementPlanNotFound),
		errors.Is(err, service.ErrEntitlementCustomerNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.As(err, &valErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}
