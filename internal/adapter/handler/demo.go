package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// DemoHandler exposes POST /auth/demo — the zero-friction sandbox entry
// (docs/spec_demo_mode.md D2). Registered ONLY when DEMO_MODE=true; in
// every other deployment the route does not exist (404).
type DemoHandler struct {
	demo   *service.DemoService
	auth   *service.AuthService
	secure bool
}

func NewDemoHandler(demo *service.DemoService, auth *service.AuthService, secure bool) *DemoHandler {
	return &DemoHandler{demo: demo, auth: auth, secure: secure}
}

// StartSession logs the caller in as the seeded demo user and sets the
// standard session cookie — the same cookie path as a password login, so
// everything downstream (dashboard, RBAC, audit actor) behaves normally.
func (h *DemoHandler) StartSession(c *gin.Context) {
	ctx := c.Request.Context()
	user, err := h.demo.DemoUser(ctx)
	if err != nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "demo environment is still warming up — try again in a few seconds")
		return
	}
	token, err := h.auth.OpenSessionForUser(ctx, user, c.Request.UserAgent())
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to open demo session")
		return
	}
	maxAge := int(h.auth.SessionTTL().Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(domain.SessionCookieName, token, maxAge, "/", "", h.secure, true)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"demo":    true,
			"email":   service.DemoUserEmail,
			"api_key": service.DemoAPIKey,
		},
	})
}
