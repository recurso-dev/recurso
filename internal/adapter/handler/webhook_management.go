package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	var req CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	input := service.CreateEndpointInput{
		TenantID: tenantID.(uuid.UUID),
		URL:      req.URL,
		Events:   req.Events,
	}

	endpoint, err := h.webhookService.CreateEndpoint(c.Request.Context(), input)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	// Return with secret on create (only time it's visible)
	c.JSON(http.StatusCreated, gin.H{"data": toWebhookEndpointResponse(endpoint, true)})
}

// ListEndpoints returns all webhook endpoints for the tenant
func (h *WebhookManagementHandler) ListEndpoints(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	endpoints, err := h.webhookService.ListEndpoints(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	var response []WebhookEndpointResponse
	for _, e := range endpoints {
		response = append(response, toWebhookEndpointResponse(e, false))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateEndpointStatus pauses ("inactive") or resumes ("active") an endpoint.
// Paused endpoints stop receiving deliveries but keep their secret and config.
func (h *WebhookManagementHandler) UpdateEndpointStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid id")
		return
	}
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != "active" && req.Status != "inactive") {
		respondError(c, http.StatusBadRequest, codeValidationFailed, `status must be "active" or "inactive"`)
		return
	}
	if err := h.webhookService.UpdateEndpointStatus(c.Request.Context(), tenantID.(uuid.UUID), id, req.Status); err != nil {
		if errors.Is(err, service.ErrEndpointNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, "endpoint not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

// DeleteEndpoint deletes a webhook endpoint
func (h *WebhookManagementHandler) DeleteEndpoint(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid id")
		return
	}

	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	if err := h.webhookService.DeleteEndpoint(c.Request.Context(), tenantID.(uuid.UUID), id); err != nil {
		if errors.Is(err, service.ErrEndpointNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, "endpoint not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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

// EventDeliveryResponse is the API response for a webhook delivery attempt
type EventDeliveryResponse struct {
	ID                string `json:"id"`
	EventID           string `json:"event_id"`
	WebhookEndpointID string `json:"webhook_endpoint_id"`
	EndpointURL       string `json:"endpoint_url,omitempty"`
	Status            string `json:"status"`
	Attempts          int    `json:"attempts"`
	LastStatusCode    int    `json:"last_status_code,omitempty"`
	LastError         string `json:"last_error,omitempty"`
	NextRetryAt       string `json:"next_retry_at,omitempty"`
	DeliveredAt       string `json:"delivered_at,omitempty"`
	CreatedAt         string `json:"created_at"`
}

func toEventDeliveryResponse(d *domain.EventDelivery, endpointURL string) EventDeliveryResponse {
	resp := EventDeliveryResponse{
		ID:                d.ID.String(),
		EventID:           d.EventID.String(),
		WebhookEndpointID: d.WebhookEndpointID.String(),
		EndpointURL:       endpointURL,
		Status:            d.DeliveryStatus(),
		Attempts:          d.Attempt,
		LastStatusCode:    d.StatusCode,
		CreatedAt:         d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	// response_body holds the failure reason recorded by the delivery worker
	// (transport error or "HTTP <code>: <body>"). On success it holds the
	// receiver's 2xx body, which is not an error — omit it.
	if resp.Status != domain.DeliveryStatusSucceeded {
		resp.LastError = d.ResponseBody
	}
	if d.NextRetryAt != nil {
		resp.NextRetryAt = d.NextRetryAt.Format("2006-01-02T15:04:05Z")
	}
	if d.DeliveredAt != nil {
		resp.DeliveredAt = d.DeliveredAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

// respondWebhookServiceError maps webhook service errors to the error envelope.
func respondWebhookServiceError(c *gin.Context, err error) {
	switch err {
	case service.ErrEventNotFound, service.ErrEndpointNotFound:
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case service.ErrInvalidDeliveryStatus:
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}

// ListEventDeliveries returns delivery attempts for an event across endpoints
func (h *WebhookManagementHandler) ListEventDeliveries(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid id")
		return
	}

	details, err := h.webhookService.ListEventDeliveries(c.Request.Context(), tenantID.(uuid.UUID), eventID)
	if err != nil {
		respondWebhookServiceError(c, err)
		return
	}

	response := make([]EventDeliveryResponse, 0, len(details))
	for _, d := range details {
		response = append(response, toEventDeliveryResponse(d.Delivery, d.EndpointURL))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// ListEndpointDeliveries returns recent deliveries for a webhook endpoint
func (h *WebhookManagementHandler) ListEndpointDeliveries(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	endpointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid id")
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

	deliveries, endpoint, err := h.webhookService.ListEndpointDeliveries(
		c.Request.Context(), tenantID.(uuid.UUID), endpointID, c.Query("status"), limit, offset)
	if err != nil {
		respondWebhookServiceError(c, err)
		return
	}

	response := make([]EventDeliveryResponse, 0, len(deliveries))
	for _, d := range deliveries {
		response = append(response, toEventDeliveryResponse(d, endpoint.URL))
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// RedeliverEvent re-enqueues delivery of an event to its subscribed endpoints
func (h *WebhookManagementHandler) RedeliverEvent(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid id")
		return
	}

	queued, err := h.webhookService.RedeliverEvent(c.Request.Context(), tenantID.(uuid.UUID), eventID)
	if err != nil {
		respondWebhookServiceError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{
		"event_id":          eventID.String(),
		"deliveries_queued": queued,
	}})
}
