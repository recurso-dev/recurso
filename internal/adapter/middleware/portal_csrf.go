package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
)

// PortalCSRFCookieName is the double-submit CSRF cookie. Unlike portal_session
// it is deliberately NOT httpOnly: the portal page JS reads it and echoes the
// value back in the X-CSRF-Token header, and the server checks the two match.
// An attacker on another origin can't read this cookie (same-origin policy), so
// they can't forge the matching header — this is the defense-in-depth backstop
// behind the session cookie's SameSite=Lax.
const PortalCSRFCookieName = "portal_csrf"

// portalCSRFHeader is the header the client echoes the cookie value in. It is
// already advertised in the CORS Access-Control-Allow-Headers list.
const portalCSRFHeader = "X-CSRF-Token"

// SetPortalCSRFCookie issues a fresh double-submit token. Called at login (so a
// session is immediately ready to make state-changing calls) and lazily by the
// middleware on the first authenticated safe request of any pre-existing session
// that predates this cookie (so older sessions self-heal without re-login).
// SameSite=Lax + Secure mirror the session cookie; path=/ so it rides every
// portal request.
func SetPortalCSRFCookie(c *gin.Context, token string, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	// maxAge mirrors the 7-day session; httpOnly=false so the page JS can read it.
	c.SetCookie(PortalCSRFCookieName, token, 60*60*24*7, "/", "", secure, false)
}

// PortalCSRFMiddleware enforces double-submit CSRF on the authenticated portal
// API. It runs AFTER PortalAuthMiddleware, so it only ever issues tokens to a
// validated session.
//
//   - Safe methods (GET/HEAD/OPTIONS) never change state, so they pass; if the
//     session has no CSRF cookie yet (a pre-existing login, or a fresh one that
//     hasn't been issued a token), one is minted here so the subsequent
//     state-changing request has a token to echo.
//   - State-changing methods (POST/PUT/PATCH/DELETE) require the X-CSRF-Token
//     header to be present and constant-time-equal to the portal_csrf cookie.
//     Missing or mismatched → 403.
func PortalCSRFMiddleware(secure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, _ := c.Cookie(PortalCSRFCookieName)

		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			if cookie == "" {
				SetPortalCSRFCookie(c, db.GenerateSecureToken(), secure)
			}
			c.Next()
		default:
			header := c.GetHeader(portalCSRFHeader)
			if cookie == "" || header == "" ||
				subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
				httperr.Abort(c, http.StatusForbidden, httperr.CodeForbidden,
					"missing or invalid CSRF token")
				return
			}
			c.Next()
		}
	}
}
