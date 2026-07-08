package recurso

import "fmt"

// APIError is returned for any non-2xx response from the Recurso API. It
// decodes the standard error envelope ({"error": {"code", "message"}}) and
// carries the HTTP status code.
type APIError struct {
	// Code is the stable, machine-readable error code (e.g. "NOT_FOUND",
	// "VALIDATION_ERROR"). It may be empty for non-JSON error bodies.
	Code string
	// Message is the human-readable explanation.
	Message string
	// StatusCode is the HTTP status code of the response.
	StatusCode int
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("recurso: %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("recurso: %d: %s", e.StatusCode, e.Message)
}

// errorEnvelope is the wire shape of an API error body.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
