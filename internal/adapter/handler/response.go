package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard API error codes
const (
	ErrCodeBadRequest    = "BAD_REQUEST"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeConflict      = "CONFLICT"
	ErrCodeInternalError = "INTERNAL_ERROR"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeInvalidAPIKey = "INVALID_API_KEY"
	ErrCodeRateLimited   = "RATE_LIMITED"
)

// APIError represents a standardized error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondError sends a standardized error response
func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": APIError{
			Code:    code,
			Message: message,
		},
	})
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
