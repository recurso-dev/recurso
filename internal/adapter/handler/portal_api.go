package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/service"
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "valid email required")
		return
	}

	link, err := h.portalService.RequestMagicLink(c.Request.Context(), req.Email)
	if err != nil {
		if err == service.ErrCustomerNotFound {
			// Don't reveal if email exists - security best practice
			c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a login link has been sent"})
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	resp := gin.H{"message": "Login link sent to your email"}
	// Expose the link in the response only in development; in production the
	// token must travel exclusively via email.
	if os.Getenv("APP_ENV") == "development" {
		resp["_dev_link"] = "/portal/verify?token=" + link.Token
	}
	c.JSON(http.StatusOK, resp)
}

// VerifyMagicLink verifies the magic link and creates a session
func (h *PortalAPIHandler) VerifyMagicLink(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "token required")
		return
	}

	session, err := h.portalService.VerifyMagicLink(c.Request.Context(), token)
	if err != nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	invoices, err := h.portalService.GetCustomerInvoices(c.Request.Context(), customerID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

// GetProfile returns the customer's profile
func (h *PortalAPIHandler) GetProfile(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	customer, err := h.portalService.GetCustomer(c.Request.Context(), customerID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, customer)
}

// portalUpdatePaymentMethodRequest carries the gateway-tokenized card metadata.
// No raw PAN is ever accepted — these fields come from the client-side gateway
// tokenization, exactly like the admin flow. The customer is resolved from the
// session, never from the body.
type portalUpdatePaymentMethodRequest struct {
	CardBrand string `json:"card_brand" binding:"required"`
	CardLast4 string `json:"card_last4" binding:"required,len=4"`
	ExpMonth  int    `json:"card_exp_month" binding:"required,min=1,max=12"`
	ExpYear   int    `json:"card_exp_year" binding:"required,min=2020"`
}

// UpdatePaymentMethod updates the authenticated portal customer's payment method.
func (h *PortalAPIHandler) UpdatePaymentMethod(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	var req portalUpdatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if err := h.portalService.UpdatePaymentMethod(
		c.Request.Context(), customerID.(uuid.UUID),
		req.CardBrand, req.CardLast4, req.ExpMonth, req.ExpYear,
	); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// portalDisputeRequest is the body for raising an invoice dispute/query.
type portalDisputeRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RaiseDispute lets the authenticated customer raise a dispute/query on one of
// their own invoices. The invoice id comes from the path; ownership is enforced
// server-side.
func (h *PortalAPIHandler) RaiseDispute(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}

	var req portalDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "reason required")
		return
	}

	dispute, err := h.portalService.RaiseDispute(c.Request.Context(), customerID.(uuid.UUID), invoiceID, req.Reason)
	if err != nil {
		if err == service.ErrInvoiceNotFound {
			respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dispute})
}

// GetDisputes returns the authenticated customer's disputes.
func (h *PortalAPIHandler) GetDisputes(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	disputes, err := h.portalService.GetCustomerDisputes(c.Request.Context(), customerID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": disputes})
}

type PortalRedeemGiftRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemGift handles gift redemption
func (h *PortalAPIHandler) RedeemGift(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	var req PortalRedeemGiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "code required")
		return
	}

	if err := h.portalService.RedeemGift(c.Request.Context(), customerID.(uuid.UUID), req.Code); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gift redeemed successfully"})
}

// Logout invalidates the session
func (h *PortalAPIHandler) Logout(c *gin.Context) {
	c.SetCookie("portal_session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
