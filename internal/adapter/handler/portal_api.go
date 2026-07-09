package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

// customerPaymentStore is the customer persistence the portal card-update flow
// needs. Satisfied by *db.CustomerRepository (kept as a narrow local interface
// so we don't widen port.CustomerRepository and break its many test mocks).
type customerPaymentStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	GetStripeCustomerID(ctx context.Context, id uuid.UUID) (string, error)
	SetStripeCustomerID(ctx context.Context, id uuid.UUID, stripeCustomerID string) error
	SetDefaultPaymentMethod(ctx context.Context, id uuid.UUID, paymentMethodID, brand, last4 string, expMonth, expYear int) error
}

// paymentMethodSetup is the Stripe capability for saving a reusable card via a
// SetupIntent. Satisfied by *gateway.StripeGateway; nil when Stripe isn't
// configured, which disables the card-update endpoints.
type paymentMethodSetup interface {
	EnsureStripeCustomer(ctx context.Context, existingID, email, name string) (string, error)
	CreateSetupIntent(ctx context.Context, stripeCustomerID string, metadata map[string]string) (string, error)
	FinalizeSetupIntent(ctx context.Context, setupIntentID string) (*port.SavedCard, error)
}

// PortalAPIHandler handles customer-facing portal endpoints
type PortalAPIHandler struct {
	portalService  *service.PortalService
	customerStore  customerPaymentStore
	paymentSetup   paymentMethodSetup
	publishableKey string
}

func NewPortalAPIHandler(portalService *service.PortalService) *PortalAPIHandler {
	return &PortalAPIHandler{portalService: portalService}
}

// SetPaymentMethodSetup wires the Stripe SetupIntent card-update flow (ENG-5).
// When either dependency is nil the card-update endpoints report that
// self-serve card update isn't available on the deployment.
func (h *PortalAPIHandler) SetPaymentMethodSetup(store customerPaymentStore, setup paymentMethodSetup, publishableKey string) {
	h.customerStore = store
	h.paymentSetup = setup
	h.publishableKey = publishableKey
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

// StartPaymentMethodSetup creates a Stripe SetupIntent for the authenticated
// portal customer and returns the client_secret the Payment Element confirms,
// plus the publishable key. Card data goes browser->Stripe directly — Recurso
// never sees the PAN (PCI SAQ-A preserved).
func (h *PortalAPIHandler) StartPaymentMethodSetup(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	if h.paymentSetup == nil || h.customerStore == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "self-serve card update isn't available on this deployment")
		return
	}
	id := customerID.(uuid.UUID)
	ctx := c.Request.Context()

	cust, err := h.customerStore.GetByID(ctx, id)
	if err != nil || cust == nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load customer")
		return
	}
	existing, err := h.customerStore.GetStripeCustomerID(ctx, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load customer")
		return
	}
	name := ""
	if cust.Name != nil {
		name = *cust.Name
	}

	stripeCustomerID, err := h.paymentSetup.EnsureStripeCustomer(ctx, existing, cust.Email, name)
	if err != nil {
		respondError(c, http.StatusBadGateway, codeInternalError, "failed to prepare payment setup")
		return
	}
	if stripeCustomerID != existing {
		if err := h.customerStore.SetStripeCustomerID(ctx, id, stripeCustomerID); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to save customer")
			return
		}
	}

	clientSecret, err := h.paymentSetup.CreateSetupIntent(ctx, stripeCustomerID, map[string]string{"customer_id": id.String()})
	if err != nil {
		respondError(c, http.StatusBadGateway, codeInternalError, "failed to start payment setup")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"client_secret":   clientSecret,
		"publishable_key": h.publishableKey,
	}})
}

// ConfirmPaymentMethod finalizes a confirmed SetupIntent: it verifies the saved
// method belongs to the authenticated customer, records it as the default, and
// refreshes the displayed card. The frontend sends the setup_intent id after
// stripe.confirmSetup succeeds.
func (h *PortalAPIHandler) ConfirmPaymentMethod(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	if h.paymentSetup == nil || h.customerStore == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "self-serve card update isn't available on this deployment")
		return
	}
	var req struct {
		SetupIntentID string `json:"setup_intent_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "setup_intent_id required")
		return
	}
	id := customerID.(uuid.UUID)
	ctx := c.Request.Context()

	saved, err := h.paymentSetup.FinalizeSetupIntent(ctx, req.SetupIntentID)
	if err != nil {
		respondError(c, http.StatusBadGateway, codeInternalError, "failed to verify payment method")
		return
	}
	// Bind to the authenticated customer — a SetupIntent whose metadata points
	// at a different customer must never update this one's card.
	if saved.CustomerID != id.String() {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "payment method does not belong to this customer")
		return
	}
	if saved.Status != "succeeded" || saved.PaymentMethodID == "" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "processing"}})
		return
	}

	if err := h.customerStore.SetDefaultPaymentMethod(ctx, id, saved.PaymentMethodID, saved.Brand, saved.Last4, saved.ExpMonth, saved.ExpYear); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to save payment method")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"status": "saved",
		"card": gin.H{
			"brand":     saved.Brand,
			"last4":     saved.Last4,
			"exp_month": saved.ExpMonth,
			"exp_year":  saved.ExpYear,
		},
	}})
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
