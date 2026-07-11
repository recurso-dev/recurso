package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/swapnull-in/recur-so/internal/adapter/httperr"
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
