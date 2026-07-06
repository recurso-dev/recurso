package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swapnull-in/recur-so/internal/adapter/httperr"
)

// Standard API error codes (snake_case machine codes; see httperr package).
const (
	ErrCodeBadRequest    = httperr.CodeValidationFailed
	ErrCodeUnauthorized  = httperr.CodeUnauthorized
	ErrCodeNotFound      = httperr.CodeNotFound
	ErrCodeConflict      = httperr.CodeConflict
	ErrCodeInternalError = httperr.CodeInternalError
	ErrCodeValidation    = httperr.CodeValidationFailed
	ErrCodeInvalidAPIKey = httperr.CodeInvalidAPIKey
	ErrCodeRateLimited   = httperr.CodeRateLimited
)

// APIError represents a standardized error response.
type APIError = httperr.APIError

// RespondError sends a standardized error response in the canonical shape
// {"error": {"code": "...", "message": "..."}}.
func RespondError(c *gin.Context, status int, code, message string) {
	httperr.Respond(c, status, code, message)
}

// RespondBadRequest sends a 400 error
func RespondBadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, ErrCodeBadRequest, message)
}

// RespondNotFound sends a 404 error
func RespondNotFound(c *gin.Context, message string) {
	RespondError(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// RespondInternalError sends a 500 error
func RespondInternalError(c *gin.Context, message string) {
	RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, message)
}

// RespondValidationError sends a 422 error
func RespondValidationError(c *gin.Context, message string) {
	RespondError(c, http.StatusUnprocessableEntity, ErrCodeValidation, message)
}

// RespondSuccess sends a standardized success response with data
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// RespondCreated sends a 201 response with data
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// RespondList sends a paginated list response
func RespondList(c *gin.Context, data interface{}, total, page, perPage int) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}
