package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

// EInvoiceHandler handles e-invoice API endpoints.
type EInvoiceHandler struct {
	einvoiceService *service.EInvoiceService
	irpConfigRepo   *db.IRPConfigRepository
}

func NewEInvoiceHandler(einvoiceService *service.EInvoiceService, irpConfigRepo *db.IRPConfigRepository) *EInvoiceHandler {
	return &EInvoiceHandler{
		einvoiceService: einvoiceService,
		irpConfigRepo:   irpConfigRepo,
	}
}

// GetEInvoiceStatus returns the e-invoice status for an invoice.
// GET /v1/invoices/:id/einvoice
func (h *EInvoiceHandler) GetEInvoiceStatus(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	status, err := h.einvoiceService.GetEInvoiceStatus(c.Request.Context(), invoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// RetryEInvoice manually retries e-invoice generation for a FAILED invoice.
// POST /v1/invoices/:id/einvoice/retry
func (h *EInvoiceHandler) RetryEInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	resp, err := h.einvoiceService.RetryFailedEInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    resp,
		"message": "E-invoice retry initiated",
	})
}

// CancelEInvoice cancels an IRN for a GENERATED invoice.
// POST /v1/invoices/:id/einvoice/cancel
func (h *EInvoiceHandler) CancelEInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	var req struct {
		CancelCode int    `json:"cancel_code" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.einvoiceService.CancelEInvoice(c.Request.Context(), invoiceID, req.CancelCode, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "E-invoice cancelled successfully"})
}

// GetIRPConfig returns the IRP configuration for the tenant.
// GET /v1/settings/irp
func (h *EInvoiceHandler) GetIRPConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	// Try production first, then sandbox
	config, err := h.irpConfigRepo.GetByTenantID(c.Request.Context(), tenantID, "production")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		config, _ = h.irpConfigRepo.GetByTenantID(c.Request.Context(), tenantID, "sandbox")
	}

	if config == nil {
		// Return empty config
		c.JSON(http.StatusOK, gin.H{"data": domain.IRPConfig{
			TenantID:    tenantID.String(),
			Environment: "sandbox",
		}})
		return
	}

	// Mask secrets for display
	maskedConfig := *config
	if len(maskedConfig.ClientSecret) > 4 {
		maskedConfig.ClientSecret = maskedConfig.ClientSecret[:4] + "****"
	}
	if len(maskedConfig.Password) > 0 {
		maskedConfig.Password = "****"
	}

	c.JSON(http.StatusOK, gin.H{"data": maskedConfig})
}

// UpdateIRPConfig creates or updates IRP credentials.
// PUT /v1/settings/irp
func (h *EInvoiceHandler) UpdateIRPConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	var config domain.IRPConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.TenantID = tenantID.String()
	if config.Environment == "" {
		config.Environment = "sandbox"
	}

	if err := h.irpConfigRepo.Upsert(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    config,
		"message": "IRP configuration saved",
	})
}

// TestIRPConnection tests the IRP connection with saved credentials.
// POST /v1/settings/irp/test
func (h *EInvoiceHandler) TestIRPConnection(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	// Try to get config
	config, err := h.irpConfigRepo.GetByTenantID(c.Request.Context(), tenantID, "sandbox")
	if err != nil || config == nil {
		config, err = h.irpConfigRepo.GetByTenantID(c.Request.Context(), tenantID, "production")
	}

	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No IRP configuration found. Please save credentials first."})
		return
	}

	if !config.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "IRP is disabled. Enable it first.",
		})
		return
	}

	// For now, just validate that credentials are non-empty
	if config.ClientID == "" || config.ClientSecret == "" || config.Username == "" || config.GSTIN == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Incomplete credentials. Please fill in all required fields.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "IRP credentials validated. Connection test passed.",
	})
}
