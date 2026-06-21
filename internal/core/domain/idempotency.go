package domain

import "time"

// StoredResponse represents a cached API response for idempotency
type StoredResponse struct {
	Status    int               `json:"status"`
	Body      []byte            `json:"body"`
	Headers   map[string]string `json:"headers"`
	CreatedAt time.Time         `json:"created_at"`
}
