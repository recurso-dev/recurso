package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

// WebhookManagementHandler handles webhook endpoint and event management
type WebhookManagementHandler struct {
	webhookService *service.WebhookService
}

func NewWebhookManagementHandler(webhookService *service.WebhookService) *WebhookManagementHandler {
	return &WebhookManagementHandler{webhookService: webhookService}
}

// CreateEndpointRequest represents the request body for creating a webhook endpoint
type CreateEndpointRequest struct {
	URL    string   `json:"url" binding:"required"`
	Events []string `json:"events" binding:"required"`
}

// WebhookEndpointResponse is the API response for a webhook endpoint
type WebhookEndpointResponse struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"` // Only returned on create
	Events    []string `json:"events"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
}

func toWebhookEndpointResponse(e *domain.WebhookEndpoint, includeSecret bool) WebhookEndpointResponse {
	resp := WebhookEndpointResponse{
		ID:        e.ID.String(),
		URL:       e.URL,
		Events:    e.Events,
		Status:    e.Status,
		CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if includeSecret {
		resp.Secret = e.Secret
	}
	return resp
}

// CreateEndpoint creates a new webhook endpoint
func (h *WebhookManagementHandler) CreateEndpoint(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := service.CreateEndpointInput{
		TenantID: tenantID.(uuid.UUID),
		URL:      req.URL,
		Events:   req.Events,
	}

	endpoint, err := h.webhookService.CreateEndpoint(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return with secret on create (only time it's visible)
	c.JSON(http.StatusCreated, gin.H{"data": toWebhookEndpointResponse(endpoint, true)})
}

// ListEndpoints returns all webhook endpoints for the tenant
func (h *WebhookManagementHandler) ListEndpoints(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpoints, err := h.webhookService.ListEndpoints(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []WebhookEndpointResponse
	for _, e := range endpoints {
		response = append(response, toWebhookEndpointResponse(e, false))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// DeleteEndpoint deletes a webhook endpoint
func (h *WebhookManagementHandler) DeleteEndpoint(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.webhookService.DeleteEndpoint(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// EventResponse is the API response for an event
type EventResponse struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  string                 `json:"created_at"`
}

func toEventResponse(e *domain.Event) EventResponse {
	return EventResponse{
		ID:         e.ID.String(),
		Type:       e.Type,
		ObjectType: e.ObjectType,
		ObjectID:   e.ObjectID.String(),
		Data:       e.Data,
		CreatedAt:  e.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ListEvents returns events for the tenant
func (h *WebhookManagementHandler) ListEvents(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	events, err := h.webhookService.ListEvents(c.Request.Context(), tenantID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []EventResponse
	for _, e := range events {
		response = append(response, toEventResponse(e))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// GetEventTypes returns supported event types
func (h *WebhookManagementHandler) GetEventTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": domain.AllEventTypes()})
}
