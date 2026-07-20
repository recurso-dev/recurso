package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// ConsentHandler handles consent API endpoints
type ConsentHandler struct {
	consentService *service.ConsentService
}

// NewConsentHandler creates a new ConsentHandler
func NewConsentHandler(consentService *service.ConsentService) *ConsentHandler {
	return &ConsentHandler{consentService: consentService}
}

// RecordConsentRequest is the request body for recording consent
type RecordConsentRequest struct {
	CustomerID     string `json:"customer_id" binding:"required"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	ConsentType    string `json:"consent_type" binding:"required"`
	Granted        bool   `json:"granted" binding:"required"`
	ConsentText    string `json:"consent_text,omitempty"`
}

// RecordConsent handles POST /v1/consents
func (h *ConsentHandler) RecordConsent(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "Unauthorized")
		return
	}

	var req RecordConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid customer ID")
		return
	}

	var subscriptionID *uuid.UUID
	if req.SubscriptionID != "" {
		id, err := uuid.Parse(req.SubscriptionID)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid subscription ID")
			return
		}
		subscriptionID = &id
	}

	// Map consent type
	var consentType domain.ConsentType
	switch req.ConsentType {
	case "recurring_billing":
		consentType = domain.ConsentTypeRecurringBilling
	case "email_marketing":
		consentType = domain.ConsentTypeEmailMarketing
	case "data_processing":
		consentType = domain.ConsentTypeDataProcessing
	case "terms_of_service":
		consentType = domain.ConsentTypeTermsOfService
	case "privacy_policy":
		consentType = domain.ConsentTypePrivacyPolicy
	default:
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid consent type")
		return
	}

	// Use default consent text for recurring billing if not provided
	consentText := req.ConsentText
	if consentText == "" && consentType == domain.ConsentTypeRecurringBilling {
		consentText = domain.RecurringBillingConsentText
	}

	record := domain.ConsentRecord{
		CustomerID:     customerID,
		SubscriptionID: subscriptionID,
		ConsentType:    consentType,
		Granted:        req.Granted,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
		ConsentText:    consentText,
		Version:        domain.CurrentConsentVersion,
	}

	consent, err := h.consentService.RecordConsent(c.Request.Context(), tenantID, record)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to record consent")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": consent})
}

// GetCustomerConsents handles GET /v1/customers/:id/consents
func (h *ConsentHandler) GetCustomerConsents(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "Unauthorized")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid customer ID")
		return
	}

	consents, err := h.consentService.GetCustomerConsents(c.Request.Context(), tenantID, customerID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to retrieve consents")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   consents,
	})
}

// RevokeConsentRequest is the request body for revoking consent
type RevokeConsentRequest struct {
	ConsentID string `json:"consent_id" binding:"required"`
}

// RevokeConsent handles POST /v1/consents/revoke
func (h *ConsentHandler) RevokeConsent(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "Unauthorized")
		return
	}

	var req RevokeConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	consentID, err := uuid.Parse(req.ConsentID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid consent ID")
		return
	}

	if err := h.consentService.RevokeConsent(c.Request.Context(), tenantID, consentID); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to revoke consent")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent revoked successfully"})
}

// GetSubscriptionConsent handles GET /v1/subscriptions/:id/consent
func (h *ConsentHandler) GetSubscriptionConsent(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "Unauthorized")
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid subscription ID")
		return
	}

	consent, err := h.consentService.GetSubscriptionConsent(c.Request.Context(), tenantID, subscriptionID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to retrieve consent")
		return
	}

	if consent == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "No consent found for this subscription")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": consent})
}
