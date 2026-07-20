package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// SSOHandler serves both the tenant-scoped admin config endpoints
// (/v1/sso/connection, owner/admin only) and the public per-tenant SP endpoints
// (/auth/saml/:tenantID/...). Public SP logins reuse the Phase 1 session path
// via AuthService.
type SSOHandler struct {
	sso          *service.SSOService
	auth         *service.AuthService
	dashboardURL string
	secure       bool
	logger       *slog.Logger
}

func NewSSOHandler(sso *service.SSOService, auth *service.AuthService, dashboardURL string, secureCookie bool) *SSOHandler {
	return &SSOHandler{
		sso:          sso,
		auth:         auth,
		dashboardURL: strings.TrimRight(dashboardURL, "/"),
		secure:       secureCookie,
		logger:       slog.Default().With("handler", "sso"),
	}
}

// --- admin config (tenant-scoped, owner/admin only) ---

// requireManager gates writes to owner/admin. API-key (machine) callers have
// full tenant access already (mirrors TeamHandler); session members get 403.
func (h *SSOHandler) requireManager(c *gin.Context) bool {
	role, hasUser := middleware.GetUserRole(c)
	if !hasUser {
		return true
	}
	if domain.Role(role).CanManageTeam() {
		return true
	}
	respondError(c, http.StatusForbidden, codeForbidden, "requires owner or admin role")
	return false
}

func (h *SSOHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	id := middleware.GetTenantID(c)
	if id == uuid.Nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, false
	}
	return id, true
}

// ssoConnectionView is the safe serialized shape of a connection. The IdP
// certificate/metadata are echoed back so the admin UI can display current
// config; no secrets beyond what the admin themselves submitted are exposed.
type ssoConnectionView struct {
	TenantID       string `json:"tenant_id"`
	IDPEntityID    string `json:"idp_entity_id"`
	IDPSSOURL      string `json:"idp_sso_url"`
	IDPCertificate string `json:"idp_certificate"`
	IDPMetadataXML string `json:"idp_metadata_xml"`
	Enabled        bool   `json:"enabled"`
	Configured     bool   `json:"configured"`
	MetadataURL    string `json:"sp_metadata_url"`
	ACSURL         string `json:"sp_acs_url"`
}

func (h *SSOHandler) toView(conn *domain.SSOConnection) ssoConnectionView {
	return ssoConnectionView{
		TenantID:       conn.TenantID.String(),
		IDPEntityID:    conn.IDPEntityID,
		IDPSSOURL:      conn.IDPSSOURL,
		IDPCertificate: conn.IDPCertificate,
		IDPMetadataXML: conn.IDPMetadataXML,
		Enabled:        conn.Enabled,
		Configured:     conn.Configured(),
		MetadataURL:    h.sso.SPMetadataURL(conn.TenantID),
		ACSURL:         h.sso.SPACSURL(conn.TenantID),
	}
}

// GET /v1/sso/connection
func (h *SSOHandler) GetConnection(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	conn, err := h.sso.GetConnection(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrSSOConnectionNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, "no SSO connection configured")
			return
		}
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.toView(conn)})
}

type upsertSSOConnectionRequest struct {
	IDPEntityID    string `json:"idp_entity_id"`
	IDPSSOURL      string `json:"idp_sso_url"`
	IDPCertificate string `json:"idp_certificate"`
	IDPMetadataXML string `json:"idp_metadata_xml"`
	Enabled        bool   `json:"enabled"`
}

// PUT /v1/sso/connection
func (h *SSOHandler) UpsertConnection(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	if !h.requireManager(c) {
		return
	}
	var req upsertSSOConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	conn, err := h.sso.UpsertConnection(c.Request.Context(), tenantID, service.UpsertConnectionInput{
		IDPMetadataXML: req.IDPMetadataXML,
		IDPEntityID:    req.IDPEntityID,
		IDPSSOURL:      req.IDPSSOURL,
		IDPCertificate: req.IDPCertificate,
		Enabled:        req.Enabled,
	})
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.toView(conn)})
}

// DELETE /v1/sso/connection
func (h *SSOHandler) DeleteConnection(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	if !h.requireManager(c) {
		return
	}
	if err := h.sso.DeleteConnection(c.Request.Context(), tenantID); err != nil {
		if errors.Is(err, domain.ErrSSOConnectionNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, "no SSO connection configured")
			return
		}
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sso connection removed"})
}

// --- public SP endpoints (per-tenant by UUID in the path) ---

func parseTenantParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("tenantID"))
	if err != nil {
		respondError(c, http.StatusNotFound, codeNotFound, "unknown tenant")
		return uuid.Nil, false
	}
	return id, true
}

// GET /auth/saml/:tenantID/metadata → SP metadata XML.
func (h *SSOHandler) Metadata(c *gin.Context) {
	tenantID, ok := parseTenantParam(c)
	if !ok {
		return
	}
	xmlBytes, err := h.sso.Metadata(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusNotFound, codeNotFound, "no SSO connection configured for this tenant")
		return
	}
	c.Data(http.StatusOK, "application/samlmetadata+xml", xmlBytes)
}

// GET /auth/saml/:tenantID/login → 302 to the IdP (AuthnRequest) when enabled,
// else 404.
func (h *SSOHandler) Login(c *gin.Context) {
	tenantID, ok := parseTenantParam(c)
	if !ok {
		return
	}
	redirectURL, err := h.sso.LoginRedirectURL(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrSSONotEnabled) {
			respondError(c, http.StatusNotFound, codeNotFound, "SSO is not enabled for this tenant")
			return
		}
		h.logger.Warn("saml login failed", "tenant_id", tenantID, "error", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to start SSO login")
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

// POST /auth/saml/:tenantID/acs → validate the SAMLResponse, map the email to an
// existing user in the tenant, open a session, and redirect to the dashboard.
func (h *SSOHandler) ACS(c *gin.Context) {
	tenantID, ok := parseTenantParam(c)
	if !ok {
		return
	}
	user, err := h.sso.ProcessACS(c.Request.Context(), tenantID, c.Request)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSSONotEnabled):
			respondError(c, http.StatusNotFound, codeNotFound, "SSO is not enabled for this tenant")
		case errors.Is(err, domain.ErrSSOUserNotFound):
			// No JIT provisioning in this phase: a validated but unknown identity
			// is a clear, actionable 403.
			respondError(c, http.StatusForbidden, codeForbidden, "no user in this tenant matches the SSO identity; ask an admin to invite you first")
		case errors.Is(err, domain.ErrSSOInvalidAssertion):
			respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid SAML assertion")
		default:
			h.logger.Warn("saml acs failed", "tenant_id", tenantID, "error", err)
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to process SSO assertion")
		}
		return
	}

	sessionToken, err := h.auth.OpenSessionForUser(c.Request.Context(), user, c.GetHeader("User-Agent"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to open session")
		return
	}
	h.setSessionCookie(c, sessionToken)
	h.logger.Info("saml login", "tenant_id", tenantID, "user_id", user.ID)
	c.Redirect(http.StatusFound, h.dashboardURL+"/")
}

// setSessionCookie mirrors AuthHandler.setSessionCookie exactly.
func (h *SSOHandler) setSessionCookie(c *gin.Context, token string) {
	maxAge := int(h.auth.SessionTTL().Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(domain.SessionCookieName, token, maxAge, "/", "", h.secure, true)
}
