package handler

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// customerPaymentStore is the customer persistence the portal card-update flow
// needs. Satisfied by *db.CustomerRepository (kept as a narrow local interface
// so we don't widen port.CustomerRepository and break its many test mocks).
// Uses GetByIDPublic (not GetByID) because the portal is public — its request
// context carries no tenant_id, which GetByID requires.
type customerPaymentStore interface {
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	GetStripeCustomerID(ctx context.Context, id uuid.UUID) (string, error)
	SetStripeCustomerID(ctx context.Context, id uuid.UUID, stripeCustomerID string) error
	SetDefaultPaymentMethod(ctx context.Context, id uuid.UUID, paymentMethodID, brand, last4 string, expMonth, expYear int, gatewayConnectionID *uuid.UUID) error
}

// paymentSetupResolver picks the Stripe gateway a card is saved on for a tenant
// (B1 autopay): the tenant's BYO connection when they have one (so recurring
// charges land in their own account), else the platform gateway. connID is the
// BYO connection's id (nil for platform) and is recorded with the saved card so
// off-session charges route to the same gateway. publishableKey is the account's
// Stripe publishable key the browser needs to confirm the SetupIntent (the BYO
// account's key, or the platform key). Satisfied by a main.go wrapper over the
// gateway resolver + connection service.
type paymentSetupResolver interface {
	SetupForTenant(ctx context.Context, tenantID uuid.UUID) (setup paymentMethodSetup, connID *uuid.UUID, publishableKey string)
}

// paymentMethodSetup is the Stripe capability for saving a reusable card via a
// SetupIntent. Satisfied by *gateway.StripeGateway; nil when Stripe isn't
// configured, which disables the card-update endpoints.
type paymentMethodSetup interface {
	EnsureStripeCustomer(ctx context.Context, existingID, email, name string) (string, error)
	CreateSetupIntent(ctx context.Context, stripeCustomerID string, metadata map[string]string) (string, error)
	FinalizeSetupIntent(ctx context.Context, setupIntentID string) (*port.SavedCard, error)
}

// mandateReauth creates a fresh UPI mandate whose AuthURL the customer approves
// on Razorpay's hosted page (ENG-5 Phase 3a). Satisfied by
// *service.MandateService; nil when Razorpay isn't really configured — the
// mock gateway's AuthURL would strand customers on a fake page.
type mandateReauth interface {
	CreateMandate(ctx context.Context, input service.CreateMandateInput) (*service.CreateMandateOutput, error)
}

// portalInvoiceReader lists a customer's invoices (newest first) so the re-auth
// flow can size the mandate cap from what they're actually billed. Satisfied by
// *db.InvoiceRepository via its customer-scoped query.
type portalInvoiceReader interface {
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error)
}

// PortalAPIHandler handles customer-facing portal endpoints
type PortalAPIHandler struct {
	portalService  *service.PortalService
	customerStore  customerPaymentStore
	paymentSetup   paymentMethodSetup
	setupResolver  paymentSetupResolver // B1: BYO-gateway routing for card save (nil-safe)
	publishableKey string
	mandateSvc     mandateReauth
	invoiceReader  portalInvoiceReader
}

// SetPaymentSetupResolver wires BYO-gateway routing for card save (B1 autopay).
// nil-safe: without it, cards save on the platform gateway (h.paymentSetup) and
// record a nil connection.
func (h *PortalAPIHandler) SetPaymentSetupResolver(r paymentSetupResolver) {
	h.setupResolver = r
}

// byoSetupResolver routes card save to a tenant's BYO Stripe gateway when they
// have an active connection with a secret, else the platform gateway (B1).
type byoSetupResolver struct {
	stripeFor      func(ctx context.Context, tenantID uuid.UUID) port.PaymentGateway
	activeConn     func(ctx context.Context, tenantID uuid.UUID) *domain.GatewayConnection
	platform       paymentMethodSetup
	platformPubKey string
}

// NewBYOSetupResolver builds the resolver. stripeFor returns the tenant's Stripe
// gateway (BYO or env); activeConn returns the tenant's active Stripe connection
// (nil = none); platform/platformPubKey are the env fallbacks.
func NewBYOSetupResolver(
	stripeFor func(ctx context.Context, tenantID uuid.UUID) port.PaymentGateway,
	activeConn func(ctx context.Context, tenantID uuid.UUID) *domain.GatewayConnection,
	platform paymentMethodSetup,
	platformPubKey string,
) *byoSetupResolver {
	return &byoSetupResolver{stripeFor: stripeFor, activeConn: activeConn, platform: platform, platformPubKey: platformPubKey}
}

func (r *byoSetupResolver) SetupForTenant(ctx context.Context, tenantID uuid.UUID) (paymentMethodSetup, *uuid.UUID, string) {
	if conn := r.activeConn(ctx, tenantID); conn != nil && conn.HasSecret() {
		if setup, ok := r.stripeFor(ctx, tenantID).(paymentMethodSetup); ok {
			return setup, &conn.ID, conn.PublicKey
		}
	}
	return r.platform, nil, r.platformPubKey
}

// resolveSetup returns the Stripe gateway a card should be saved on for the
// customer's tenant, the connection id to record (nil = platform), and the
// publishable key the browser needs. Falls back to the platform gateway when no
// resolver is wired.
func (h *PortalAPIHandler) resolveSetup(ctx context.Context, cust *domain.Customer) (paymentMethodSetup, *uuid.UUID, string) {
	if h.setupResolver != nil {
		if setup, connID, pubKey := h.setupResolver.SetupForTenant(ctx, cust.TenantID); setup != nil {
			return setup, connID, pubKey
		}
	}
	return h.paymentSetup, nil, h.publishableKey
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

// SetMandateReauth wires the UPI-mandate re-authorization flow (ENG-5 Phase
// 3a). Wire it only with real Razorpay keys; nil leaves the endpoint reporting
// unavailable.
func (h *PortalAPIHandler) SetMandateReauth(store customerPaymentStore, svc mandateReauth, invoices portalInvoiceReader) {
	if h.customerStore == nil {
		h.customerStore = store
	}
	h.mandateSvc = svc
	h.invoiceReader = invoices
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
		respondInternalError(c, err)
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

	// Set session cookie. Secure everywhere except local dev (mirrors the
	// dashboard session cookie) so the portal token isn't sent over plain HTTP.
	// SameSite=Lax (explicit, like the dashboard cookie) so the browser doesn't
	// attach it to cross-site state-changing requests — the CSRF backstop — and
	// so behavior doesn't depend on the browser's default for an unset attribute.
	secureCookie := os.Getenv("APP_ENV") != "development"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("portal_session", session.Token, 60*60*24*7, "/", "", secureCookie, true)

	// The session is delivered ONLY via the httpOnly cookie above — never in the
	// JSON body, so it is never readable by page JavaScript (XSS-safe). The
	// client authenticates by sending the cookie (credentials: "include").
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged in successfully",
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
		respondInternalError(c, err)
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
		respondInternalError(c, err)
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
		respondInternalError(c, err)
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

	cust, err := h.customerStore.GetByIDPublic(ctx, id)
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

	// B1: create the SetupIntent on the tenant's gateway (BYO or platform).
	// ConfirmPaymentMethod resolves the same gateway (GetActive is deterministic).
	setup, _, pubKey := h.resolveSetup(ctx, cust)

	stripeCustomerID, err := setup.EnsureStripeCustomer(ctx, existing, cust.Email, name)
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

	clientSecret, err := setup.CreateSetupIntent(ctx, stripeCustomerID, map[string]string{"customer_id": id.String()})
	if err != nil {
		respondError(c, http.StatusBadGateway, codeInternalError, "failed to start payment setup")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"client_secret":   clientSecret,
		"publishable_key": pubKey,
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

	// Resolve the same gateway the SetupIntent was created on (B1) so finalize
	// hits the right account, and record which connection saved the card.
	cust, err := h.customerStore.GetByIDPublic(ctx, id)
	if err != nil || cust == nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load customer")
		return
	}
	setup, connID, _ := h.resolveSetup(ctx, cust)

	saved, err := setup.FinalizeSetupIntent(ctx, req.SetupIntentID)
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

	if err := h.customerStore.SetDefaultPaymentMethod(ctx, id, saved.PaymentMethodID, saved.Brand, saved.Last4, saved.ExpMonth, saved.ExpYear, connID); err != nil {
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

// StartMandateReauth creates a fresh UPI mandate for the authenticated portal
// customer and returns the Razorpay-hosted authorization URL (ENG-5 Phase 3a).
// The cap is sized from the customer's own billing history, the frequency is
// monthly, and the mandate only activates when Razorpay confirms the token
// (token.confirmed webhook → HandleAuthorization) — nothing here trusts the
// browser.
func (h *PortalAPIHandler) StartMandateReauth(c *gin.Context) {
	customerID, exists := c.Get("portal_customer_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	if h.mandateSvc == nil || h.customerStore == nil || h.invoiceReader == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "self-serve mandate re-authorization isn't available on this deployment")
		return
	}

	// VPA is optional — without it Razorpay's hosted page collects the UPI id.
	var req struct {
		VPA string `json:"vpa"`
	}
	_ = c.ShouldBindJSON(&req)

	id := customerID.(uuid.UUID)
	ctx := c.Request.Context()

	cust, err := h.customerStore.GetByIDPublic(ctx, id)
	if err != nil || cust == nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load customer")
		return
	}

	// Cap = 2× the largest recent invoice: headroom for proration and plan
	// changes without granting an unbounded pull.
	invoices, err := h.invoiceReader.GetByCustomerID(ctx, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load billing history")
		return
	}
	var maxTotal int64
	for i, inv := range invoices {
		if i >= 12 { // newest first; a year of invoices is plenty
			break
		}
		if inv.Status == domain.InvoiceStatusVoid {
			continue
		}
		if inv.Total > maxTotal {
			maxTotal = inv.Total
		}
	}
	if maxTotal <= 0 {
		respondError(c, http.StatusConflict, codeValidationFailed, "no billing history to authorize a mandate against")
		return
	}

	// MandateService reads the customer through the tenant-scoped repo; the
	// portal request carries no tenant, so inject the customer's own (same
	// pattern as the payment webhooks).
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, cust.TenantID)
	out, err := h.mandateSvc.CreateMandate(tenantCtx, service.CreateMandateInput{
		TenantID:   cust.TenantID,
		CustomerID: id,
		VPA:        req.VPA,
		MaxAmount:  maxTotal * 2,
		Frequency:  "monthly",
	})
	if err != nil {
		if errors.Is(err, service.ErrCustomerPhoneRequired) {
			respondError(c, http.StatusUnprocessableEntity, codeValidationFailed, "a phone number is required for UPI Autopay — please ask the merchant to add one to your account")
			return
		}
		respondError(c, http.StatusBadGateway, codeInternalError, "failed to start mandate authorization")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"auth_url":   out.AuthURL,
		"mandate_id": out.Mandate.ID,
		"status":     string(out.Mandate.Status),
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
		respondInternalError(c, err)
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
		respondInternalError(c, err)
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
	// Match the SameSite attribute used when the cookie was set so the deletion
	// cookie is a reliable overwrite rather than a second, differently-scoped one.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("portal_session", "", -1, "/", "", os.Getenv("APP_ENV") != "development", true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
