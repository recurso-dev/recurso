package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type UsageHandler struct {
	repo *db.UsageRepository
}

func NewUsageHandler(repo *db.UsageRepository) *UsageHandler {
	return &UsageHandler{repo: repo}
}

type recordEventRequest struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	CustomerID     string `json:"customer_id" binding:"required"`
	Dimension      string `json:"dimension" binding:"required"`
	Quantity       int64  `json:"quantity" binding:"required"`
}

func (h *UsageHandler) RecordEvent(c *gin.Context) {
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

	// Usage requires usage_repository to be tenant-aware?
	// Currently usage_events table might NOT have tenant_id?
	// But let's inject anyway.
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if ok { // If ok, inject. If not (unauth?), usage handler is auth...
		ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
		if err := h.repo.RecordEvent(ctx, event); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record event")
			return
		}
	} else {
		// If auth middleware didn't run?
		if err := h.repo.RecordEvent(c.Request.Context(), event); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record event")
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"status": "recorded", "event_id": event.ID})
}
