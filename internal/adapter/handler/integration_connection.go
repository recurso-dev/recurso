package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// IntegrationConnectionHandler serves the tenant-scoped BYO integration
// endpoints (tax/CRM/storage). Secrets are write-only: responses carry only the
// category/provider, non-secret config, and whether secrets are set.
type IntegrationConnectionHandler struct {
	svc    *service.IntegrationConnectionService
	logger *slog.Logger
}

func NewIntegrationConnectionHandler(svc *service.IntegrationConnectionService) *IntegrationConnectionHandler {
	return &IntegrationConnectionHandler{svc: svc, logger: slog.Default().With("handler", "integration_connection")}
}

func (h *IntegrationConnectionHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	id := middleware.GetTenantID(c)
	if id == uuid.Nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, false
	}
	return id, true
}

func (h *IntegrationConnectionHandler) requireManager(c *gin.Context) bool {
	role, hasUser := middleware.GetUserRole(c)
	if !hasUser {
		return true
	}
	if domain.Role(role).CanManageTeam() {
		return true
	}
	respondError(c, http.StatusForbidden, codeForbidden, "requires owner or admin role")
	return false
}

// integrationView is the secret-free projection returned to clients.
type integrationView struct {
	ID         uuid.UUID         `json:"id"`
	Category   string            `json:"category"`
	Provider   string            `json:"provider"`
	Config     map[string]string `json:"config"` // non-secret fields only
	HasSecrets bool              `json:"has_secrets"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

func (h *IntegrationConnectionHandler) List(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	conns, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	views := make([]integrationView, 0, len(conns))
	for _, conn := range conns {
		present, hasSecret := h.svc.SafeConfig(c.Request.Context(), conn)
		views = append(views, integrationView{
			ID:         conn.ID,
			Category:   string(conn.Category),
			Provider:   conn.Provider,
			Config:     present,
			HasSecrets: hasSecret,
			CreatedAt:  conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"connections": views,
		"vault_ready": h.svc.VaultReady(),
	}})
}

type integrationConnectRequest struct {
	Category string            `json:"category" binding:"required"`
	Provider string            `json:"provider" binding:"required"`
	Config   map[string]string `json:"config" binding:"required"`
}

func (h *IntegrationConnectionHandler) Connect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok || !h.requireManager(c) {
		return
	}
	var req integrationConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	conn, err := h.svc.Connect(c.Request.Context(), tenantID, req.Category, req.Provider, req.Config)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	present, hasSecret := h.svc.SafeConfig(c.Request.Context(), conn)
	c.JSON(http.StatusCreated, gin.H{"data": integrationView{
		ID: conn.ID, Category: string(conn.Category), Provider: conn.Provider,
		Config: present, HasSecrets: hasSecret,
		CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}})
}

func (h *IntegrationConnectionHandler) Disconnect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok || !h.requireManager(c) {
		return
	}
	if err := h.svc.Disconnect(c.Request.Context(), tenantID, c.Param("category"), c.Param("provider")); err != nil {
		h.respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

func (h *IntegrationConnectionHandler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrIntegrationConnectionNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, "no active connection for that integration")
	case errors.Is(err, domain.ErrGatewayVaultUnavailable):
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "credential vault unavailable (GATEWAY_ENCRYPTION_KEY not configured)")
	case service.IsIntegrationConnectionValidationError(err):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	default:
		respondInternalError(c, err)
	}
}
