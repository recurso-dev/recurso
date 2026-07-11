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

// UsageHandler exposes the usage platform:
//
//	POST /v1/usage/events              record a metered usage event
//	GET  /v1/usage                     time-windowed usage buckets
//	GET  /v1/usage/dimensions          the tenant's dimension catalog
//	GET  /v1/subscriptions/:id/usage   current-period + lifetime usage
type UsageHandler struct {
	svc *service.UsageService
}

func NewUsageHandler(svc *service.UsageService) *UsageHandler {
	return &UsageHandler{svc: svc}
}

type recordEventRequest struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	CustomerID     string `json:"customer_id" binding:"required"`
	Dimension      string `json:"dimension" binding:"required"`
	Quantity       int64  `json:"quantity" binding:"required"`
}

func (h *UsageHandler) RecordEvent(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}

	var req recordEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	subID, err := uuid.Parse(req.SubscriptionID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid Subscription ID")
		return
	}

	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid Customer ID")
		return
	}

	event := &domain.UsageEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		CustomerID:     custID,
		Dimension:      req.Dimension,
		Quantity:       req.Quantity,
		Timestamp:      time.Now().UTC(),
	}

	if err := h.svc.RecordEvent(ctx, tenantID, event); err != nil {
		switch err {
		case service.ErrUsageSubscriptionNotFound:
			respondError(c, http.StatusNotFound, codeNotFound, "subscription not found")
		case service.ErrUsageCustomerMismatch:
			respondError(c, http.StatusBadRequest, codeValidationFailed, "customer does not match subscription")
		default:
			respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record event")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "recorded", "event_id": event.ID})
}

// QueryUsage handles GET /v1/usage — time-windowed usage buckets.
//
// Query params: subscription_id or customer_id (at least one required),
// dimension (optional), from/to (RFC3339, default: last 30 days),
// granularity (day | month, default day).
func (h *UsageHandler) QueryUsage(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}

	var params service.UsageQueryParams

	if raw := c.Query("subscription_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "subscription_id must be a valid uuid")
			return
		}
		params.SubscriptionID = &id
	}
	if raw := c.Query("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "customer_id must be a valid uuid")
			return
		}
		params.CustomerID = &id
	}
	params.Dimension = c.Query("dimension")
	params.Granularity = c.Query("granularity")

	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "from must be an RFC3339 timestamp")
			return
		}
		params.From = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "to must be an RFC3339 timestamp")
			return
		}
		params.To = &t
	}

	buckets, resolved, err := h.svc.QueryUsage(ctx, tenantID, params)
	if err != nil {
		respondUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":        buckets,
		"from":        resolved.From,
		"to":          resolved.To,
		"granularity": resolved.Granularity,
	})
}

// GetSubscriptionUsage handles GET /v1/subscriptions/:id/usage — the
// current billing period's usage per dimension plus lifetime totals, with
// entitlement limits joined in where a matching feature_key exists.
func (h *UsageHandler) GetSubscriptionUsage(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}

	subID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription id")
		return
	}

	usage, err := h.svc.GetSubscriptionUsage(ctx, tenantID, subID)
	if err != nil {
		respondUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, usage)
}

// ListDimensions handles GET /v1/usage/dimensions — the tenant's distinct
// usage dimensions with first/last seen and event counts.
func (h *UsageHandler) ListDimensions(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}

	dims, err := h.svc.ListDimensions(ctx, tenantID)
	if err != nil {
		respondUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dims})
}

// usageTenantCtx extracts the authenticated tenant and returns a request
// context carrying it for tenant-scoped repositories.
func usageTenantCtx(c *gin.Context) (uuid.UUID, context.Context, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, nil, false
	}
	return tenantID, context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

// respondUsageError maps usage service errors onto the canonical envelope.
func respondUsageError(c *gin.Context, err error) {
	var valErr service.UsageValidationError
	switch {
	case errors.Is(err, service.ErrUsageSubscriptionNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.As(err, &valErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}
