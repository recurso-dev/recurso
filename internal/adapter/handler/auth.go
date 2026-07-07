package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
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
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func toUserView(u *domain.User) userView {
	return userView{ID: u.ID.String(), Email: u.Email, Name: u.Name, Role: string(u.Role)}
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
			respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		// Generic message for BOTH unknown-email and wrong-password.
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid credentials")
		return
	}

	h.setSessionCookie(c, res.SessionToken)
	c.JSON(http.StatusOK, gin.H{
		"user":   toUserView(res.User),
		"tenant": tenantView{ID: res.Tenant.ID.String(), Name: res.Tenant.Name},
	})
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
