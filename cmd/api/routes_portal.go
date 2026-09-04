package main

import (
	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/service"
)

// portalHandlers carries every handler, middleware and service the customer
// portal route table needs: the public magic-link endpoints and the
// session-authenticated /portal/api group. main() wires dependencies and
// builds this once; the table itself lives here so main.go stays a wiring
// file. cmd/api/openapi_drift_test.go scans this file, so a dropped or renamed
// line fails CI.
type portalHandlers struct {
	pdfHandler       *handler.InvoicePDFHandler
	portalAPIHandler *handler.PortalAPIHandler
	portalService    *service.PortalService
	publicLimit      gin.HandlerFunc
	secureCookie     bool
}

// registerPortalRoutes mounts the customer portal: the public /portal/auth
// magic-link endpoints (h.publicLimit) and the /portal/api group behind the
// portal session + CSRF middleware.
func registerPortalRoutes(r *gin.Engine, h *portalHandlers) {
	// Customer Portal Auth (P25)
	r.POST("/portal/auth/request", h.publicLimit, h.portalAPIHandler.RequestMagicLink)
	r.GET("/portal/auth/verify", h.publicLimit, h.portalAPIHandler.VerifyMagicLink)  // deprecated: query-string token, kept one release for links in flight
	r.POST("/portal/auth/verify", h.publicLimit, h.portalAPIHandler.VerifyMagicLink) // preferred: token in POST body (not logged/refererred)

	// Protected Customer Portal Routes
	portal := r.Group("/portal/api")
	portal.Use(middleware.PortalAuthMiddleware(h.portalService))
	// Double-submit CSRF backstop behind the session cookie's SameSite=Lax:
	// state-changing portal calls must echo the portal_csrf cookie in the
	// X-CSRF-Token header. Runs after auth so tokens are only issued to a valid
	// session; safe GETs lazily mint the cookie so pre-existing sessions heal.
	portal.Use(middleware.PortalCSRFMiddleware(h.secureCookie))
	{
		portal.GET("/profile", h.portalAPIHandler.GetProfile)
		portal.GET("/invoices", h.portalAPIHandler.GetInvoices)
		// Customer-scoped invoice PDF (ownership-checked in the handler) so the
		// portal's Download-PDF button has a public, token-authed endpoint (ENG-152).
		portal.GET("/invoices/:id/pdf", h.pdfHandler.PortalDownloadPDF)
		portal.PUT("/payment-method", h.portalAPIHandler.UpdatePaymentMethod)
		portal.POST("/payment-method/setup-intent", h.portalAPIHandler.StartPaymentMethodSetup)
		portal.POST("/payment-method/bank-setup-intent", h.portalAPIHandler.StartBankAccountSetup) // ACH (Inc 3a)
		portal.POST("/payment-method/confirm", h.portalAPIHandler.ConfirmPaymentMethod)
		portal.POST("/payment-method/mandate", h.portalAPIHandler.StartMandateReauth)
		portal.GET("/disputes", h.portalAPIHandler.GetDisputes)
		portal.POST("/invoices/:id/dispute", h.portalAPIHandler.RaiseDispute)
		portal.POST("/redeem", h.portalAPIHandler.RedeemGift)
		portal.POST("/logout", h.portalAPIHandler.Logout)
	}
}
