package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/service"
)

// CancellationHandler handles subscription cancellation endpoints
type CancellationHandler struct {
	subscriptionService *service.SubscriptionService
	consentService      *service.ConsentService
	notificationService *service.NotificationService
}

// NewCancellationHandler creates a new CancellationHandler
func NewCancellationHandler(
	subscriptionService *service.SubscriptionService,
	consentService *service.ConsentService,
	notificationService *service.NotificationService,
) *CancellationHandler {
	return &CancellationHandler{
		subscriptionService: subscriptionService,
		consentService:      consentService,
		notificationService: notificationService,
	}
}

// CancellationReason represents available cancellation reasons
type CancellationReason string

const (
	ReasonTooExpensive    CancellationReason = "too_expensive"
	ReasonNotUsing        CancellationReason = "not_using"
	ReasonSwitching       CancellationReason = "switching_competitor"
	ReasonMissingFeatures CancellationReason = "missing_features"
	ReasonTechnicalIssues CancellationReason = "technical_issues"
	ReasonCustomerService CancellationReason = "customer_service"
	ReasonTemporaryPause  CancellationReason = "temporary_pause"
	ReasonOther           CancellationReason = "other"
)

// CancelSubscriptionRequest is the request body for cancelling a subscription
type CancelSubscriptionRequest struct {
	CancelAtPeriodEnd bool               `json:"cancel_at_period_end"`
	Immediately       bool               `json:"immediately"`
	Reason            CancellationReason `json:"reason" binding:"required"`
	Feedback          string             `json:"feedback,omitempty"`
	RevokeConsent     bool               `json:"revoke_consent"`
}

// CancelSubscriptionResponse is the response for subscription cancellation
type CancelSubscriptionResponse struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	CancelledAt       *time.Time `json:"cancelled_at,omitempty"`
	CurrentPeriodEnd  time.Time  `json:"current_period_end"`
	Reason            string     `json:"cancellation_reason"`
	Message           string     `json:"message"`
}

// CancelSubscription handles POST /v1/subscriptions/:id/cancel
func (h *CancellationHandler) CancelSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to cancel at period end if not specified
	cancelImmediately := req.Immediately
	if !req.Immediately && !req.CancelAtPeriodEnd {
		req.CancelAtPeriodEnd = true
	}

	// Cancel the subscription
	subscription, err := h.subscriptionService.Cancel(c.Request.Context(), tenantID, subscriptionID, cancelImmediately, string(req.Reason), req.Feedback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
		return
	}

	// Revoke consent if requested (best-effort: errors are ignored so the request doesn't fail)
	if req.RevokeConsent {
		_ = h.consentService.RevokeSubscriptionConsent(c.Request.Context(), tenantID, subscriptionID)
	}

	// Send cancellation confirmation email
	if h.notificationService != nil {
		accessUntil := subscription.CurrentPeriodEnd.Format("January 2, 2006")
		portalURL := c.GetHeader("X-Portal-URL")
		if portalURL == "" {
			portalURL = "https://portal.yourapp.com"
		}

		_ = h.notificationService.SendSubscriptionCancelled(
			c.Request.Context(),
			subscription.CustomerEmail,
			subscription.CustomerName,
			subscription.PlanName,
			accessUntil,
			portalURL+"/reactivate/"+subscriptionID.String(),
		)
	}

	// Prepare response
	var cancelledAt *time.Time
	if cancelImmediately {
		now := time.Now()
		cancelledAt = &now
	}

	message := "Subscription will be cancelled at the end of the billing period"
	if cancelImmediately {
		message = "Subscription has been cancelled immediately"
	}

	c.JSON(http.StatusOK, CancelSubscriptionResponse{
		ID:                subscriptionID.String(),
		Status:            subscription.Status,
		CancelAtPeriodEnd: req.CancelAtPeriodEnd,
		CancelledAt:       cancelledAt,
		CurrentPeriodEnd:  subscription.CurrentPeriodEnd,
		Reason:            string(req.Reason),
		Message:           message,
	})
}

// GetCancellationReasons handles GET /v1/cancellation-reasons
func (h *CancellationHandler) GetCancellationReasons(c *gin.Context) {
	reasons := []gin.H{
		{"id": ReasonTooExpensive, "label": "Too expensive", "allows_feedback": false},
		{"id": ReasonNotUsing, "label": "Not using enough", "allows_feedback": false},
		{"id": ReasonSwitching, "label": "Switching to competitor", "allows_feedback": true},
		{"id": ReasonMissingFeatures, "label": "Missing features I need", "allows_feedback": true},
		{"id": ReasonTechnicalIssues, "label": "Technical issues", "allows_feedback": true},
		{"id": ReasonCustomerService, "label": "Customer service experience", "allows_feedback": true},
		{"id": ReasonTemporaryPause, "label": "Just need a break", "allows_feedback": false},
		{"id": ReasonOther, "label": "Other", "allows_feedback": true},
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   reasons,
	})
}

// ReactivateSubscription handles POST /v1/subscriptions/:id/reactivate
func (h *CancellationHandler) ReactivateSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	subscription, err := h.subscriptionService.Reactivate(c.Request.Context(), tenantID, subscriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reactivate subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      subscription.ID,
		"status":  subscription.Status,
		"message": "Subscription reactivated successfully",
	})
}
