package domain

import (
	"time"

	"github.com/google/uuid"
)

// MCPSettings is a tenant's opt-in configuration for the MCP server. Tier3Enabled
// gates the money-path / destructive MCP tools (convert quote→invoice, cancel
// subscription, create credit note, wallet top-up, …). It defaults to false: a
// tenant must explicitly opt in before an agent can perform those actions.
type MCPSettings struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Tier3Enabled bool      `json:"tier3_enabled"`
	UpdatedAt    time.Time `json:"updated_at"`
}
