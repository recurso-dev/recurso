package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
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
	// Properties are optional free-form attributes; the unique aggregation
	// counts distinct values of one property (usage-based billing v1).
	Properties map[string]string `json:"properties"`
	// TransactionID is the caller's idempotency key: a retried event with
	// the same (subscription, transaction_id) collapses to the original.
	TransactionID string `json:"transaction_id"`
	// DynamicAmount is the caller-supplied exact price for this event in minor
	// units (non-negative); a `dynamic` charge bills the sum over the period.
	DynamicAmount int64 `json:"dynamic_amount"`
}

// toEvent converts the request into a domain event (uuid errors reported
// per field by the caller).
func (r recordEventRequest) toEvent() (*domain.UsageEvent, error) {
	subID, err := uuid.Parse(r.SubscriptionID)
	if err != nil {
		return nil, errInvalidSubscriptionID
	}
	custID, err := uuid.Parse(r.CustomerID)
	if err != nil {
		return nil, errInvalidCustomerID
	}
	return &domain.UsageEvent{
		ID:             uuid.New(),
		SubscriptionID: subID,
		CustomerID:     custID,
		Dimension:      r.Dimension,
		Quantity:       r.Quantity,
		Timestamp:      time.Now().UTC(),
		Properties:     r.Properties,
		TransactionID:  r.TransactionID,
		DynamicAmount:  r.DynamicAmount,
	}, nil
}

var (
	errInvalidSubscriptionID = errors.New("invalid subscription_id")
	errInvalidCustomerID     = errors.New("invalid customer_id")
)

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

	event, err := req.toEvent()
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	duplicate, err := h.svc.RecordEventIdempotent(ctx, tenantID, event)
	if err != nil {
		var valErr service.UsageValidationError
		switch {
		case errors.Is(err, service.ErrUsageSubscriptionNotFound):
			respondError(c, http.StatusNotFound, codeNotFound, "subscription not found")
		case errors.Is(err, service.ErrUsageCustomerMismatch):
			respondError(c, http.StatusBadRequest, codeValidationFailed, "customer does not match subscription")
		case errors.As(err, &valErr):
			respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
		default:
			respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record event")
		}
		return
	}

	status := "recorded"
	code := http.StatusCreated
	if duplicate {
		// Idempotent replay: the original event, not a new one (C1).
		status = "duplicate"
		code = http.StatusOK
	}
	c.JSON(code, gin.H{"status": status, "event_id": event.ID})
}

// RecordEventsBatch handles POST /v1/usage/events/batch — up to 500 events
// with per-item results; one bad event never fails the batch (C1).
func (h *UsageHandler) RecordEventsBatch(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}

	var req struct {
		Events []recordEventRequest `json:"events" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	events := make([]*domain.UsageEvent, len(req.Events))
	prefail := map[int]string{}
	for i, item := range req.Events {
		event, err := item.toEvent()
		if err != nil {
			// Keep the slot; the service skips nils via a placeholder that
			// fails validation, so indices stay aligned.
			prefail[i] = err.Error()
			events[i] = &domain.UsageEvent{ID: uuid.New()} // quantity 0 -> validation error placeholder
			continue
		}
		events[i] = event
	}

	results, err := h.svc.RecordEvents(ctx, tenantID, events)
	if err != nil {
		var valErr service.UsageValidationError
		if errors.As(err, &valErr) {
			respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record events")
		return
	}
	// Replace placeholder errors with the real parse failures.
	for i, msg := range prefail {
		results[i].Status, results[i].Error, results[i].EventID = "error", msg, ""
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
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
	c.JSON(http.StatusOK, gin.H{"data": usage})
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

// ListRecentEvents returns the newest raw usage events for the tenant —
// the Usage page's ingestion inspector. Filters: customer_id, dimension;
// paging: limit (<=200, default 50), offset.
func (h *UsageHandler) ListRecentEvents(c *gin.Context) {
	tenantID, ctx, ok := usageTenantCtx(c)
	if !ok {
		return
	}
	var customerID *uuid.UUID
	if v := c.Query("customer_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer_id")
			return
		}
		customerID = &id
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, offset = clampLimitOffset(limit, offset, 50, 200)
	events, err := h.svc.ListRecentEvents(ctx, tenantID, customerID, c.Query("dimension"), limit, offset)
	if err != nil {
		respondUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events})
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
		respondInternalError(c, err)
	}
}
