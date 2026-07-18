package domain

import (
	"time"

	"github.com/google/uuid"
)

// Append-only audit log (Lago-parity C2). Every config-grade mutation on
// the API writes one row; the table rejects UPDATE and DELETE at the
// database level (trigger), so history cannot be rewritten — only added to.

// AuditLog is one recorded mutation.
type AuditLog struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	// Actor is who performed the action: a dashboard user id, or "api_key"
	// for API-key-authenticated calls.
	Actor string `json:"actor"`
	// Action is METHOD + route template, e.g. "PUT /v1/plans/:id/charges".
	Action string `json:"action"`
	// EntityType is the first resource segment ("plans", "wallets", ...);
	// EntityID the :id path param when present.
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id,omitempty"`
	// Status is the response HTTP status (only 2xx mutations are recorded).
	Status int `json:"status"`
	// RequestBody is the JSON request payload, truncated to a cap — config
	// payloads, never card/credential data (those routes are excluded).
	RequestBody string    `json:"request_body,omitempty"`
	IP          string    `json:"ip,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLogFilter narrows the audit-log listing.
type AuditLogFilter struct {
	EntityType string
	EntityID   string
	Actor      string
	From       time.Time
	To         time.Time
	Limit      int
	Offset     int
}
