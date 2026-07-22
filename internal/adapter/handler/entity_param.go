package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// entityIDParam parses an optional ?entity_id= query parameter used by the
// per-entity tax-config settings endpoints (Multi-Entity Books Inc 3b). An empty
// value resolves to nil — the tenant's primary entity / default config. An
// invalid UUID returns ok=false so the caller can reject the request.
func entityIDParam(c *gin.Context) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query("entity_id"))
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &id, true
}
