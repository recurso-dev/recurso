// Package httperr defines the canonical JSON error envelope for the HTTP API.
//
// Canonical error shape (the ONLY error shape the API emits):
//
//	{"error": {"code": "<machine_code>", "message": "<human message>"}}
//
// Codes are stable snake_case machine identifiers; messages are
// human-readable and may change without notice.
//
// NOTE for cmd/api/openapi.yaml maintainers: the Error schema's oneOf
// (bare string | code+message object) can collapse to the single object
// shape documented above — the API no longer emits bare-string errors.
package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard machine-readable error codes.
const (
	CodeValidationFailed = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeConflict         = "conflict"
	CodeRateLimited      = "rate_limited"
	CodeInternalError    = "internal_error"
	CodeInvalidAPIKey    = "invalid_api_key"
	CodeKeyModeMismatch  = "key_mode_mismatch"

	// Domain-specific codes.
	CodeOverRefund         = "over_refund"
	CodeInvoiceNotPaid     = "invoice_not_paid"
	CodeInvoiceAlreadyPaid = "invoice_already_paid"
)

// APIError is the inner object of the canonical error envelope.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// envelope wraps an APIError in the canonical {"error": {...}} shape.
type envelope struct {
	Error APIError `json:"error"`
}

// Respond writes the canonical error envelope with the given status.
func Respond(c *gin.Context, status int, code, message string) {
	c.JSON(status, envelope{Error: APIError{Code: code, Message: message}})
}

// Abort writes the canonical error envelope and aborts the handler chain.
// Intended for middleware.
func Abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, envelope{Error: APIError{Code: code, Message: message}})
}

// CodeForStatus maps an HTTP status to the default machine code. Use it when
// the status is computed at runtime and no more specific code applies.
func CodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return CodeValidationFailed
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeRateLimited
	default:
		return CodeInternalError
	}
}
