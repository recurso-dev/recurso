package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/service"
)

// PortalAPIHandler handles customer-facing portal endpoints
type PortalAPIHandler struct {
	portalService *service.PortalService
}

func NewPortalAPIHandler(portalService *service.PortalService) *PortalAPIHandler {
	return &PortalAPIHandler{portalService: portalService}
}

// RequestMagicLinkRequest represents the request body
type RequestMagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestMagicLink sends a magic link to the customer's email
func (h *PortalAPIHandler) RequestMagicLink(c *gin.Context) {
	var req RequestMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid email required"})
		return
	}

	link, err := h.portalService.RequestMagicLink(c.Request.Context(), req.Email)
	if err != nil {
		if err == service.ErrCustomerNotFound {
			// Don't reveal if email exists - security best practice
			c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a login link has been sent"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// In development, return the link for testing
	// In production, this would just return a success message
	c.JSON(http.StatusOK, gin.H{
		"message":   "Login link sent to your email",
		"_dev_link": "/portal/verify?token=" + link.Token, // Remove in production
	})
}

// VerifyMagicLink verifies the magic link and creates a session
func (h *PortalAPIHandler) VerifyMagicLink(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	session, err := h.portalService.VerifyMagicLink(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set session cookie
	c.SetCookie("portal_session", session.Token, 60*60*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Logged in successfully",
		"session_token": session.Token,
	})
}

// GetInvoices returns the customer's invoices
func (h *PortalAPIHandler) GetInvoices(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	invoices, err := h.portalService.GetCustomerInvoices(c.Request.Context(), customerID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

// GetProfile returns the customer's profile
func (h *PortalAPIHandler) GetProfile(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	customer, err := h.portalService.GetCustomer(c.Request.Context(), customerID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, customer)
}

type PortalRedeemGiftRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemGift handles gift redemption
func (h *PortalAPIHandler) RedeemGift(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req PortalRedeemGiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code required"})
		return
	}

	if err := h.portalService.RedeemGift(c.Request.Context(), customerID.(uuid.UUID), req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gift redeemed successfully"})
}

// Logout invalidates the session
func (h *PortalAPIHandler) Logout(c *gin.Context) {
	c.SetCookie("portal_session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
