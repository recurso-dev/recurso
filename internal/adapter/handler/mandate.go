package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

type MandateHandler struct {
	service *service.MandateService
}

func NewMandateHandler(s *service.MandateService) *MandateHandler {
	return &MandateHandler{service: s}
}

type createMandateRequest struct {
	CustomerID     string `json:"customer_id" binding:"required"`
	SubscriptionID string `json:"subscription_id"`
	VPA            string `json:"vpa" binding:"required"`
	MaxAmount      int64  `json:"max_amount" binding:"required,gt=0"`
	Frequency      string `json:"frequency" binding:"required,oneof=weekly monthly quarterly yearly"`
}

func (h *MandateHandler) CreateMandate(c *gin.Context) {
	var req createMandateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer_id")
		return
	}

	input := service.CreateMandateInput{
		TenantID:   tenantID,
		CustomerID: customerID,
		VPA:        req.VPA,
		MaxAmount:  req.MaxAmount,
		Frequency:  req.Frequency,
	}

	if req.SubscriptionID != "" {
		subID, err := uuid.Parse(req.SubscriptionID)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription_id")
			return
		}
		input.SubscriptionID = &subID
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	result, err := h.service.CreateMandate(ctx, input)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *MandateHandler) ListMandates(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	mandates, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	if mandates == nil {
		mandates = []*domain.Mandate{}
	}

	c.JSON(http.StatusOK, gin.H{"data": mandates})
}

func (h *MandateHandler) GetMandate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid mandate id")
		return
	}

	mandate, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusNotFound, codeNotFound, "mandate not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": mandate})
}

func (h *MandateHandler) RevokeMandate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid mandate id")
		return
	}

	if err := h.service.Revoke(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}
