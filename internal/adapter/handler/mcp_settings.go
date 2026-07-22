package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// mcpSettingsStore is the persistence the handler needs; satisfied by
// *db.MCPSettingsRepository.
type mcpSettingsStore interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.MCPSettings, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, tier3Enabled bool) error
}

// MCPSettingsHandler manages a tenant's MCP server opt-in — currently just the
// Tier-3 (money-path) toggle that the MCP server checks before allowing an agent
// to perform a destructive action.
type MCPSettingsHandler struct {
	repo mcpSettingsStore
}

func NewMCPSettingsHandler(repo mcpSettingsStore) *MCPSettingsHandler {
	return &MCPSettingsHandler{repo: repo}
}

// MCPSettingsDTO is the request/response shape for the MCP settings.
type MCPSettingsDTO struct {
	// Tier3Enabled turns on the money-path/destructive MCP tools for this tenant.
	Tier3Enabled bool `json:"tier3_enabled"`
}

// GetMCPSettings returns the tenant's MCP settings, defaulting to disabled when
// none is set.
func (h *MCPSettingsHandler) GetMCPSettings(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	s, err := h.repo.GetByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	dto := MCPSettingsDTO{}
	if s != nil {
		dto.Tier3Enabled = s.Tier3Enabled
	}
	c.JSON(http.StatusOK, gin.H{"data": dto})
}

// UpdateMCPSettings upserts the tenant's MCP settings. Enabling Tier-3 is a
// deliberate, auditable opt-in to letting agents run money-path actions.
func (h *MCPSettingsHandler) UpdateMCPSettings(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	var in MCPSettingsDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if err := h.repo.Upsert(c.Request.Context(), tenantID, in.Tier3Enabled); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": in})
}
