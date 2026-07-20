package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// AuthHandler serves the public authentication endpoints (register, login,
// logout, me) that issue and consume the recurso_session cookie.
type AuthHandler struct {
	auth   *service.AuthService
	secure bool // Secure cookie attribute (true unless APP_ENV=development)
}

// NewAuthHandler builds the handler. secureCookie should be false only in
// development (so cookies work over plain http://localhost).
func NewAuthHandler(auth *service.AuthService, secureCookie bool) *AuthHandler {
	return &AuthHandler{auth: auth, secure: secureCookie}
}

// setSessionCookie writes the opaque session token as an httpOnly cookie.
// Attributes: httpOnly, Secure (outside development), SameSite=Lax, Path=/.
func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	maxAge := int(h.auth.SessionTTL().Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(domain.SessionCookieName, token, maxAge, "/", "", h.secure, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(domain.SessionCookieName, "", -1, "/", "", h.secure, true)
}

// userView is the safe, serialized shape of a user (no password hash).
type userView struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	MFAEnabled bool   `json:"mfa_enabled"`
	CreatedAt  string `json:"created_at"`
}

func toUserView(u *domain.User) userView {
	return userView{
		ID:         u.ID.String(),
		Email:      u.Email,
		Name:       u.Name,
		Role:       string(u.Role),
		MFAEnabled: u.MFAEnabled,
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
	}
}

type tenantView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- POST /auth/register ---

type authRegisterRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req authRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	res, err := h.auth.Register(c.Request.Context(), req.CompanyName, req.Name, req.Email, req.Password, c.GetHeader("User-Agent"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDuplicateEmail):
			respondError(c, http.StatusConflict, codeConflict, err.Error())
		case errors.Is(err, domain.ErrWeakPassword):
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		default:
			respondInternalError(c, err)
		}
		return
	}

	h.setSessionCookie(c, res.SessionToken)
	c.JSON(http.StatusCreated, gin.H{
		"tenant":  res.Tenant,
		"api_key": res.APIKey.KeyValue,
		"user":    toUserView(res.User),
	})
}

// --- POST /auth/login ---

type authLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req authLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Do not leak which field failed beyond generic validation.
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	res, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, c.GetHeader("User-Agent"))
	if err != nil {
		if errors.Is(err, domain.ErrAccountLocked) {
			respondError(c, http.StatusTooManyRequests, codeRateLimited, "too many failed attempts; try again later")
			return
		}
		// Generic message for BOTH unknown-email and wrong-password.
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid credentials")
		return
	}

	// MFA gate: password was correct but a second factor is required. No session
	// cookie is set; the client must call /auth/login/mfa with mfa_token + code.
	if res.MFARequired {
		c.JSON(http.StatusOK, gin.H{
			"mfa_required": true,
			"mfa_token":    res.MFAToken,
		})
		return
	}

	h.setSessionCookie(c, res.SessionToken)
	c.JSON(http.StatusOK, gin.H{
		"user":   toUserView(res.User),
		"tenant": tenantView{ID: res.Tenant.ID.String(), Name: res.Tenant.Name},
	})
}

// --- POST /auth/login/mfa ---

type authLoginMFARequest struct {
	MFAToken string `json:"mfa_token" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// LoginMFA completes a two-step login by exchanging the challenge token plus a
// TOTP or backup code for a full session cookie.
func (h *AuthHandler) LoginMFA(c *gin.Context) {
	var req authLoginMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	res, err := h.auth.LoginMFA(c.Request.Context(), req.MFAToken, req.Code, c.GetHeader("User-Agent"))
	if err != nil {
		if errors.Is(err, domain.ErrAccountLocked) {
			respondError(c, http.StatusTooManyRequests, codeRateLimited, "too many failed attempts; try again later")
			return
		}
		// Generic: never distinguish a bad code from an expired/used challenge.
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid credentials")
		return
	}

	h.setSessionCookie(c, res.SessionToken)
	c.JSON(http.StatusOK, gin.H{
		"user":   toUserView(res.User),
		"tenant": tenantView{ID: res.Tenant.ID.String(), Name: res.Tenant.Name},
	})
}

// --- POST /auth/forgot-password ---

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// genericResetMessage is returned for every forgot-password call regardless of
// whether the account exists, to prevent account enumeration.
const genericResetMessage = "If an account with that email exists, a password reset link has been sent."

// ForgotPassword always responds 200 with a generic message. If the account
// exists a single-use token is created and a reset link emailed.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Even a malformed/absent email gets the generic 200-style answer via a
		// 400 only for structural validation; do not reveal account state.
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if err := h.auth.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Log-worthy infra failure, but the client still gets the generic answer.
		slog.Default().Error("password reset request failed", "error", err)
	}
	c.JSON(http.StatusOK, gin.H{"message": genericResetMessage})
}

// --- POST /auth/reset-password ---

type resetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ResetPassword consumes a valid reset token, sets the new password, and kills
// all of the user's sessions. Invalid/expired/used tokens get a generic 400.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "password has been reset"})
	case errors.Is(err, domain.ErrWeakPassword):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	case errors.Is(err, domain.ErrInvalidResetToken):
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid or expired reset token")
	default:
		respondInternalError(c, err)
	}
}

// --- POST /auth/logout ---

func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(domain.SessionCookieName); err == nil && token != "" {
		_ = h.auth.Logout(c.Request.Context(), token)
	}
	h.clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// --- GET /auth/me ---

func (h *AuthHandler) Me(c *gin.Context) {
	token, err := c.Cookie(domain.SessionCookieName)
	if err != nil || token == "" {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "not authenticated")
		return
	}
	user, tenant, err := h.auth.Me(c.Request.Context(), token)
	if err != nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "not authenticated")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":   toUserView(user),
		"tenant": tenantView{ID: tenant.ID.String(), Name: tenant.Name},
	})
}
