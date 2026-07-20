package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
)

// Canonical API error shape (see internal/adapter/httperr):
//
//	{"error": {"code": "<machine_code>", "message": "<human message>"}}
//
// Every handler error response MUST go through respondError (or the
// httperr package) so the envelope stays consistent. Success shapes are
// unchanged by this convention.
//
// NOTE for cmd/api/openapi.yaml maintainers: the Error schema's oneOf can
// collapse to this single object shape — bare-string errors are gone.

// Local aliases so swept handler call sites stay short.
const (
	codeValidationFailed   = httperr.CodeValidationFailed
	codeUnauthorized       = httperr.CodeUnauthorized
	codeForbidden          = httperr.CodeForbidden
	codeNotFound           = httperr.CodeNotFound
	codeConflict           = httperr.CodeConflict
	codeInternalError      = httperr.CodeInternalError
	codeRateLimited        = httperr.CodeRateLimited
	codeInvoiceAlreadyPaid = httperr.CodeInvoiceAlreadyPaid
)

// respondError writes the canonical error envelope.
func respondError(c *gin.Context, status int, code, message string) {
	httperr.Respond(c, status, code, message)
}

// respondErrorStatus writes the canonical envelope, deriving the machine
// code from the (runtime-computed) HTTP status.
func respondErrorStatus(c *gin.Context, status int, message string) {
	httperr.Respond(c, status, httperr.CodeForStatus(status), message)
}

// respondInternalError is the ONLY correct way to answer a 500: the real
// error goes to the log (with method+path for correlation), the client gets
// a fixed message. Echoing err.Error() to clients leaked SQL/driver detail
// from 116 call sites before this existed — never reintroduce that.
func respondInternalError(c *gin.Context, err error) {
	slog.Error("internal error",
		"method", c.Request.Method, "path", c.FullPath(), "error", err)
	httperr.Respond(c, http.StatusInternalServerError, httperr.CodeInternalError, "internal error")
}
