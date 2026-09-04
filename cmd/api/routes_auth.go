package main

import (
	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
)

// authHandlers carries every handler and middleware the dashboard /auth route
// table needs: credential login, MFA, session state, password reset, email
// verification, OAuth social login and SAML SSO. main() wires dependencies and
// builds this once; the table itself lives here so main.go stays a wiring
// file. cmd/api/openapi_drift_test.go scans this file, so a dropped or renamed
// line fails CI.
type authHandlers struct {
	authHandler  *handler.AuthHandler
	demoHandler  *handler.DemoHandler // nil outside DEMO_MODE: /auth/demo is then not registered
	oauthHandler *handler.OAuthHandler
	publicLimit  gin.HandlerFunc
	sessionLimit gin.HandlerFunc
	ssoHandler   *handler.SSOHandler
}

// registerAuthRoutes mounts every /auth route. All are public (they establish
// or inspect the session rather than requiring one); brute-forceable endpoints
// carry h.publicLimit, per-page-load session endpoints h.sessionLimit.
func registerAuthRoutes(r *gin.Engine, h *authHandlers) {
	// Dashboard auth (public): register creates tenant + owner user + session;
	// login/logout/me operate purely on the recurso_session cookie.
	r.POST("/auth/register", h.publicLimit, h.authHandler.Register)
	r.POST("/auth/login", h.publicLimit, h.authHandler.Login)
	if h.demoHandler != nil {
		r.POST("/auth/demo", h.publicLimit, h.demoHandler.StartSession) // sandbox entry (404 outside DEMO_MODE)
	}
	r.POST("/auth/login/mfa", h.publicLimit, h.authHandler.LoginMFA)
	r.POST("/auth/logout", h.sessionLimit, h.authHandler.Logout)
	r.GET("/auth/me", h.sessionLimit, h.authHandler.Me)
	// Password reset (public): forgot-password always answers generically; the
	// reset itself consumes a single-use emailed token.
	r.POST("/auth/forgot-password", h.publicLimit, h.authHandler.ForgotPassword)
	r.POST("/auth/reset-password", h.publicLimit, h.authHandler.ResetPassword)
	// Email verification: verify-email consumes the emailed token (public);
	// resend re-issues a link to the logged-in user (session-gated).
	r.POST("/auth/verify-email", h.publicLimit, h.authHandler.VerifyEmail)
	r.POST("/auth/verify-email/resend", h.sessionLimit, h.authHandler.ResendVerification)

	// OAuth social login (public). /providers reflects which providers are
	// configured; /start issues the CSRF-state + PKCE cookie and redirects to
	// the provider; /callback validates, find-or-creates a user, opens a session
	// and redirects to the dashboard. Disabled/unknown providers 404.
	r.GET("/auth/oauth/providers", h.sessionLimit, h.oauthHandler.Providers)
	r.GET("/auth/oauth/:provider/start", h.publicLimit, h.oauthHandler.Start)
	r.GET("/auth/oauth/:provider/callback", h.publicLimit, h.oauthHandler.Callback)

	// SAML SSO SP endpoints (public, per-tenant by UUID). metadata renders the
	// SP descriptor; login 302s to the IdP when enabled; acs consumes the
	// SAMLResponse, maps to an existing tenant user (no JIT), opens a session.
	r.GET("/auth/saml/:tenantID/metadata", h.publicLimit, h.ssoHandler.Metadata)
	r.GET("/auth/saml/:tenantID/login", h.publicLimit, h.ssoHandler.Login)
	r.POST("/auth/saml/:tenantID/acs", h.publicLimit, h.ssoHandler.ACS)
}
