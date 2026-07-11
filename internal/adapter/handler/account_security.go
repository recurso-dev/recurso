package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// These endpoints live under /v1/auth and operate on the authenticated
// dashboard user. They are served by AuthHandler (which already holds the
// AuthService). API-key (machine) callers have no user context and are rejected.

// currentUser resolves the logged-in dashboard user (id + tenant) from the
// dual-auth middleware context. It returns false and writes a 401 for API-key
// callers, which have no per-user identity.
func (h *AuthHandler) currentUser(c *gin.Context) (userID, tenantID uuid.UUID, ok bool) {
	userID = middleware.GetUserID(c)
	tenantID = middleware.GetTenantID(c)
	if userID == uuid.Nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "this endpoint requires a logged-in user session")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, tenantID, true
}

// mapMFAError translates MFA domain errors to HTTP responses.
func mapMFAError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrMFAAlreadyEnabled), errors.Is(err, domain.ErrMFANotEnabled),
		errors.Is(err, domain.ErrMFANotConfigured):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidMFACode):
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid code")
	case errors.Is(err, domain.ErrUserNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, "user not found")
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}

// --- POST /v1/auth/mfa/setup ---

// MFASetup generates a pending TOTP secret and returns provisioning data for a
// QR code. MFA is not yet enabled.
func (h *AuthHandler) MFASetup(c *gin.Context) {
	userID, tenantID, ok := h.currentUser(c)
	if !ok {
		return
	}
	res, err := h.auth.SetupMFA(c.Request.Context(), tenantID, userID)
	if err != nil {
		mapMFAError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"secret":      res.Secret,
		"otpauth_url": res.OtpauthURL,
	})
}

// --- POST /v1/auth/mfa/verify ---

type mfaCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// MFAVerify confirms a TOTP code against the pending secret, enables MFA, and
// returns freshly generated one-time backup codes (shown exactly once).
func (h *AuthHandler) MFAVerify(c *gin.Context) {
	userID, tenantID, ok := h.currentUser(c)
	if !ok {
		return
	}
	var req mfaCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	codes, err := h.auth.VerifyAndEnableMFA(c.Request.Context(), tenantID, userID, req.Code)
	if err != nil {
		mapMFAError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mfa_enabled":  true,
		"backup_codes": codes,
	})
}

// --- POST /v1/auth/mfa/disable ---

// MFADisable verifies a TOTP or backup code and disables MFA.
func (h *AuthHandler) MFADisable(c *gin.Context) {
	userID, tenantID, ok := h.currentUser(c)
	if !ok {
		return
	}
	var req mfaCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if err := h.auth.DisableMFA(c.Request.Context(), tenantID, userID, req.Code); err != nil {
		mapMFAError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mfa_enabled": false})
}

// --- Session management (/v1/auth/sessions) ---

type sessionView struct {
	ID        string `json:"id"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	Current   bool   `json:"current"`
}

// ListSessions returns the user's active sessions with a flag on the current one.
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID, _, ok := h.currentUser(c)
	if !ok {
		return
	}
	current, _ := c.Cookie(domain.SessionCookieName)
	sessions, err := h.auth.ListActiveSessions(c.Request.Context(), userID, current)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	views := make([]sessionView, 0, len(sessions))
	for _, s := range sessions {
		views = append(views, sessionView{
			ID:        s.Session.ID.String(),
			UserAgent: s.Session.UserAgent,
			CreatedAt: s.Session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			ExpiresAt: s.Session.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			Current:   s.Current,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// RevokeSession deletes one of the user's own sessions. 404 if it is not theirs.
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID, _, ok := h.currentUser(c)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid session id")
		return
	}
	if err := h.auth.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, "session not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

// RevokeOtherSessions deletes all of the user's sessions except the current one.
func (h *AuthHandler) RevokeOtherSessions(c *gin.Context) {
	userID, _, ok := h.currentUser(c)
	if !ok {
		return
	}
	current, err := c.Cookie(domain.SessionCookieName)
	if err != nil || current == "" {
		// Only a live session cookie can identify "this" session to keep.
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "a session cookie is required")
		return
	}
	if err := h.auth.RevokeOtherSessions(c.Request.Context(), userID, current); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "other sessions revoked"})
}
