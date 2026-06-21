package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription_id"})
			return
		}
		input.SubscriptionID = &subID
	}

	result, err := h.service.CreateMandate(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *MandateHandler) ListMandates(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	mandates, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mandate id"})
		return
	}

	mandate, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mandate not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": mandate})
}

func (h *MandateHandler) RevokeMandate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mandate id"})
		return
	}

	if err := h.service.Revoke(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}
