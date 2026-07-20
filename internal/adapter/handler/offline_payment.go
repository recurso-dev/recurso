package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type OfflinePaymentHandler struct {
	service *service.OfflinePaymentService
}

func NewOfflinePaymentHandler(s *service.OfflinePaymentService) *OfflinePaymentHandler {
	return &OfflinePaymentHandler{service: s}
}

type createVirtualAccountRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	InvoiceID  string `json:"invoice_id"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

func (h *OfflinePaymentHandler) CreateVirtualAccount(c *gin.Context) {
	var req createVirtualAccountRequest
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

	input := service.CreateVirtualAccountInput{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     req.Amount,
	}

	if req.InvoiceID != "" {
		invoiceID, err := uuid.Parse(req.InvoiceID)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice_id")
			return
		}
		input.InvoiceID = &invoiceID
	}

	va, err := h.service.CreateVirtualAccount(c.Request.Context(), input)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": va})
}

func (h *OfflinePaymentHandler) ListVirtualAccounts(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	accounts, err := h.service.ListVirtualAccounts(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if accounts == nil {
		accounts = []*domain.VirtualAccount{}
	}

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

type recordOfflinePaymentRequest struct {
	CustomerID      string `json:"customer_id" binding:"required"`
	InvoiceID       string `json:"invoice_id"`
	PaymentType     string `json:"payment_type" binding:"required,oneof=bank_transfer cash cheque"`
	Amount          int64  `json:"amount" binding:"required,gt=0"`
	TDSAmount       int64  `json:"tds_amount" binding:"omitempty,gte=0"` // tax deducted at source by the customer; requires invoice_id
	Currency        string `json:"currency"`
	ReferenceNumber string `json:"reference_number"`
	Notes           string `json:"notes"`
	RecordedBy      string `json:"recorded_by"`
}

func (h *OfflinePaymentHandler) RecordOfflinePayment(c *gin.Context) {
	var req recordOfflinePaymentRequest
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

	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}

	input := service.RecordOfflinePaymentInput{
		TenantID:        tenantID,
		CustomerID:      customerID,
		PaymentType:     req.PaymentType,
		Amount:          req.Amount,
		TDSAmount:       req.TDSAmount,
		Currency:        currency,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
		RecordedBy:      req.RecordedBy,
	}

	if req.InvoiceID != "" {
		invoiceID, err := uuid.Parse(req.InvoiceID)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice_id")
			return
		}
		input.InvoiceID = &invoiceID
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	payment, err := h.service.RecordOfflinePayment(ctx, input)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": payment})
}

func (h *OfflinePaymentHandler) ListOfflinePayments(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	payments, err := h.service.ListOfflinePayments(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if payments == nil {
		payments = []*domain.OfflinePayment{}
	}

	c.JSON(http.StatusOK, gin.H{"data": payments})
}
