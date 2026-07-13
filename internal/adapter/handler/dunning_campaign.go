package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// respondStepError maps a step-write error: a cross-tenant (or missing)
// campaign/step surfaces as sql.ErrNoRows and is reported as 404.
func respondStepError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		respondError(c, http.StatusNotFound, codeNotFound, "campaign or step not found")
		return
	}
	respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
}

type DunningCampaignHandler struct {
	service *service.DunningCampaignService
}

func NewDunningCampaignHandler(s *service.DunningCampaignService) *DunningCampaignHandler {
	return &DunningCampaignHandler{service: s}
}

// ListCampaigns lists all dunning campaigns for the tenant
func (h *DunningCampaignHandler) ListCampaigns(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	campaigns, err := h.service.ListCampaigns(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, campaigns)
}

type createCampaignRequest struct {
	Name         string `json:"name" binding:"required"`
	TriggerEvent string `json:"trigger_event" binding:"required"`
}

// CreateCampaign creates a new dunning campaign
func (h *DunningCampaignHandler) CreateCampaign(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	now := time.Now().UTC()
	campaign := &domain.DunningCampaign{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         req.Name,
		IsActive:     true,
		TriggerEvent: req.TriggerEvent,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.service.CreateCampaign(c.Request.Context(), campaign); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

// GetCampaign returns a dunning campaign with its steps
func (h *DunningCampaignHandler) GetCampaign(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid campaign id")
		return
	}

	campaign, err := h.service.GetCampaignByID(c.Request.Context(), id, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if campaign == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "campaign not found")
		return
	}

	c.JSON(http.StatusOK, campaign)
}

type updateCampaignRequest struct {
	Name         string `json:"name"`
	IsActive     *bool  `json:"is_active"`
	TriggerEvent string `json:"trigger_event"`
}

// UpdateCampaign updates a dunning campaign
func (h *DunningCampaignHandler) UpdateCampaign(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid campaign id")
		return
	}

	var req updateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	campaign, err := h.service.GetCampaignByID(c.Request.Context(), id, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if campaign == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "campaign not found")
		return
	}

	if req.Name != "" {
		campaign.Name = req.Name
	}
	if req.IsActive != nil {
		campaign.IsActive = *req.IsActive
	}
	if req.TriggerEvent != "" {
		campaign.TriggerEvent = req.TriggerEvent
	}
	campaign.UpdatedAt = time.Now().UTC()

	if err := h.service.UpdateCampaign(c.Request.Context(), campaign); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, campaign)
}

type createCampaignStepRequest struct {
	StepOrder     int    `json:"step_order" binding:"required"`
	Channel       string `json:"channel" binding:"required"`
	DelayHours    int    `json:"delay_hours"`
	TemplateName  string `json:"template_name"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	IsPaymentWall bool   `json:"is_payment_wall"`
}

// CreateStep adds a step to a dunning campaign
func (h *DunningCampaignHandler) CreateStep(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	campaignID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid campaign id")
		return
	}

	var req createCampaignStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	step := &domain.DunningCampaignStep{
		ID:            uuid.New(),
		CampaignID:    campaignID,
		StepOrder:     req.StepOrder,
		Channel:       domain.DunningChannel(req.Channel),
		DelayHours:    req.DelayHours,
		TemplateName:  req.TemplateName,
		Subject:       req.Subject,
		Body:          req.Body,
		IsPaymentWall: req.IsPaymentWall,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.service.CreateStep(c.Request.Context(), step, tenantID); err != nil {
		respondStepError(c, err)
		return
	}

	c.JSON(http.StatusCreated, step)
}

type updateCampaignStepRequest struct {
	StepOrder     *int   `json:"step_order"`
	Channel       string `json:"channel"`
	DelayHours    *int   `json:"delay_hours"`
	TemplateName  string `json:"template_name"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	IsPaymentWall *bool  `json:"is_payment_wall"`
}

// UpdateStep updates a dunning campaign step
func (h *DunningCampaignHandler) UpdateStep(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid step id")
		return
	}

	var req updateCampaignStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	step := &domain.DunningCampaignStep{ID: id}
	if req.StepOrder != nil {
		step.StepOrder = *req.StepOrder
	}
	if req.Channel != "" {
		step.Channel = domain.DunningChannel(req.Channel)
	}
	if req.DelayHours != nil {
		step.DelayHours = *req.DelayHours
	}
	step.TemplateName = req.TemplateName
	step.Subject = req.Subject
	step.Body = req.Body
	if req.IsPaymentWall != nil {
		step.IsPaymentWall = *req.IsPaymentWall
	}

	if err := h.service.UpdateStep(c.Request.Context(), step, tenantID); err != nil {
		respondStepError(c, err)
		return
	}

	c.JSON(http.StatusOK, step)
}

// DeleteStep removes a step from a dunning campaign
func (h *DunningCampaignHandler) DeleteStep(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid step id")
		return
	}

	if err := h.service.DeleteStep(c.Request.Context(), id, tenantID); err != nil {
		respondStepError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// GetPaymentWallStatus returns the payment wall status for an invoice
func (h *DunningCampaignHandler) GetPaymentWallStatus(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}

	active, err := h.service.GetPaymentWallStatus(c.Request.Context(), invoiceID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoice_id": invoiceID, "payment_wall_active": active})
}
