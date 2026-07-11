package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type CancelFlowHandler struct {
	service *service.CancelFlowService
}

func NewCancelFlowHandler(s *service.CancelFlowService) *CancelFlowHandler {
	return &CancelFlowHandler{service: s}
}

// ListFlows lists all cancel flows for the tenant
func (h *CancelFlowHandler) ListFlows(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	flows, err := h.service.ListFlows(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, flows)
}

type createFlowRequest struct {
	Name         string `json:"name" binding:"required"`
	IsDefault    bool   `json:"is_default"`
	CooldownDays int    `json:"cooldown_days"`
}

// CreateFlow creates a new cancel flow
func (h *CancelFlowHandler) CreateFlow(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req createFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	cooldownDays := req.CooldownDays
	if cooldownDays == 0 {
		cooldownDays = 30
	}

	now := time.Now().UTC()
	flow := &domain.CancelFlow{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         req.Name,
		IsActive:     true,
		IsDefault:    req.IsDefault,
		CooldownDays: cooldownDays,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.service.CreateFlow(c.Request.Context(), flow); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, flow)
}

// GetFlow returns a cancel flow with its steps
func (h *CancelFlowHandler) GetFlow(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid flow id")
		return
	}

	flow, err := h.service.GetFlowByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if flow == nil || flow.TenantID != tenantID {
		respondError(c, http.StatusNotFound, codeNotFound, "flow not found")
		return
	}

	c.JSON(http.StatusOK, flow)
}

type updateFlowRequest struct {
	Name         string `json:"name"`
	IsActive     *bool  `json:"is_active"`
	IsDefault    *bool  `json:"is_default"`
	CooldownDays *int   `json:"cooldown_days"`
}

// UpdateFlow updates a cancel flow
func (h *CancelFlowHandler) UpdateFlow(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid flow id")
		return
	}

	var req updateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	flow, err := h.service.GetFlowByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if flow == nil || flow.TenantID != tenantID {
		respondError(c, http.StatusNotFound, codeNotFound, "flow not found")
		return
	}

	if req.Name != "" {
		flow.Name = req.Name
	}
	if req.IsActive != nil {
		flow.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		flow.IsDefault = *req.IsDefault
	}
	if req.CooldownDays != nil {
		flow.CooldownDays = *req.CooldownDays
	}
	flow.UpdatedAt = time.Now().UTC()

	if err := h.service.UpdateFlow(c.Request.Context(), flow); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, flow)
}

type createStepRequest struct {
	StepOrder int                       `json:"step_order" binding:"required"`
	StepType  domain.CancelFlowStepType `json:"step_type" binding:"required"`
	Config    json.RawMessage           `json:"config"`
}

// CreateStep adds a step to a cancel flow
func (h *CancelFlowHandler) CreateStep(c *gin.Context) {
	flowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid flow id")
		return
	}

	var req createStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	config := req.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	step := &domain.CancelFlowStep{
		ID:        uuid.New(),
		FlowID:    flowID,
		StepOrder: req.StepOrder,
		StepType:  req.StepType,
		Config:    config,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.service.CreateStep(c.Request.Context(), step); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, step)
}

type updateStepRequest struct {
	StepOrder *int                       `json:"step_order"`
	StepType  *domain.CancelFlowStepType `json:"step_type"`
	Config    json.RawMessage            `json:"config"`
}

// UpdateStep updates a cancel flow step
func (h *CancelFlowHandler) UpdateStep(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid step id")
		return
	}

	var req updateStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	step := &domain.CancelFlowStep{ID: id}
	if req.StepOrder != nil {
		step.StepOrder = *req.StepOrder
	}
	if req.StepType != nil {
		step.StepType = *req.StepType
	}
	if req.Config != nil {
		step.Config = req.Config
	}

	if err := h.service.UpdateStep(c.Request.Context(), step); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, step)
}

// DeleteStep removes a step from a cancel flow
func (h *CancelFlowHandler) DeleteStep(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid step id")
		return
	}

	if err := h.service.DeleteStep(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

type startSessionRequest struct {
	CustomerID     string `json:"customer_id" binding:"required,uuid"`
	SubscriptionID string `json:"subscription_id" binding:"required,uuid"`
}

// StartSession starts a new cancel flow session
func (h *CancelFlowHandler) StartSession(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req startSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	result, err := h.service.StartSession(c.Request.Context(), service.StartSessionInput{
		TenantID:       tenantID,
		CustomerID:     uuid.MustParse(req.CustomerID),
		SubscriptionID: uuid.MustParse(req.SubscriptionID),
	})
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusCreated, result)
}

type submitStepRequest struct {
	StepIndex int             `json:"step_index"`
	Response  json.RawMessage `json:"response" binding:"required"`
}

// SubmitStep submits a step response in a cancel flow session
func (h *CancelFlowHandler) SubmitStep(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid session id")
		return
	}

	var req submitStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	result, err := h.service.SubmitStep(c.Request.Context(), service.SubmitStepInput{
		TenantID:  tenantID,
		SessionID: sessionID,
		StepIndex: req.StepIndex,
		Response:  req.Response,
	})
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSession returns a cancel flow session
func (h *CancelFlowHandler) GetSession(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid session id")
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if session == nil || session.TenantID != tenantID {
		respondError(c, http.StatusNotFound, codeNotFound, "session not found")
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetStats returns cancel flow analytics
func (h *CancelFlowHandler) GetStats(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	flowIDStr := c.Query("flow_id")
	if flowIDStr == "" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "flow_id query parameter required")
		return
	}

	flowID, err := uuid.Parse(flowIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid flow_id")
		return
	}

	stats, err := h.service.GetFlowStats(c.Request.Context(), tenantID, flowID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, stats)
}
