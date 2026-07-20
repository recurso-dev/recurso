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

// GatewayConnectionHandler serves the tenant-scoped BYO payment-gateway
// connection endpoints (owner/admin only for writes). Secrets are write-only:
// the API never returns a stored secret, only the non-sensitive public key,
// mode, and whether a webhook secret has been set.
type GatewayConnectionHandler struct {
	svc    *service.GatewayConnectionService
	logger *slog.Logger
}

func NewGatewayConnectionHandler(svc *service.GatewayConnectionService) *GatewayConnectionHandler {
	return &GatewayConnectionHandler{svc: svc, logger: slog.Default().With("handler", "gateway_connection")}
}

func (h *GatewayConnectionHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	id := middleware.GetTenantID(c)
	if id == uuid.Nil {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, false
	}
	return id, true
}

// requireManager gates writes to owner/admin; API-key (machine) callers pass.
func (h *GatewayConnectionHandler) requireManager(c *gin.Context) bool {
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

// connectionView is the safe, secret-free projection returned to clients.
type connectionView struct {
	ID          uuid.UUID `json:"id"`
	Provider    string    `json:"provider"`
	Mode        string    `json:"mode"`
	PublicKey   string    `json:"public_key"`
	HasWebhook  bool      `json:"has_webhook_secret"`
	WebhookPath string    `json:"webhook_path"` // append to the API origin for the gateway console
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func toView(c *domain.GatewayConnection) connectionView {
	return connectionView{
		ID:          c.ID,
		Provider:    string(c.Provider),
		Mode:        string(c.Mode),
		PublicKey:   c.PublicKey,
		HasWebhook:  c.WebhookSecretEnc != "",
		WebhookPath: "/webhooks/" + string(c.Provider) + "/" + c.ID.String(),
		CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List returns the tenant's active gateway connections plus whether the
// credential vault is available (no GATEWAY_ENCRYPTION_KEY => can't connect).
func (h *GatewayConnectionHandler) List(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	conns, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	views := make([]connectionView, 0, len(conns))
	for _, conn := range conns {
		views = append(views, toView(conn))
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"connections": views,
		"vault_ready": h.svc.VaultReady(),
	}})
}

type connectRequest struct {
	Provider      string `json:"provider" binding:"required"`
	Mode          string `json:"mode"`
	PublicKey     string `json:"public_key"`
	SecretKey     string `json:"secret_key" binding:"required"`
	WebhookSecret string `json:"webhook_secret"`
}

// Connect stores (sealed) a tenant's gateway credentials, replacing any
// existing active connection for that provider.
func (h *GatewayConnectionHandler) Connect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok || !h.requireManager(c) {
		return
	}
	var req connectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	conn, err := h.svc.Connect(c.Request.Context(), tenantID, service.ConnectInput{
		Provider:      req.Provider,
		Mode:          req.Mode,
		PublicKey:     req.PublicKey,
		SecretKey:     req.SecretKey,
		WebhookSecret: req.WebhookSecret,
	})
	if err != nil {
		h.respondConnErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toView(conn)})
}

type webhookSecretRequest struct {
	WebhookSecret string `json:"webhook_secret"`
}

// SetWebhookSecret updates the webhook signing secret on the active connection
// in place (two-step connect: create the webhook in the gateway console using
// the per-connection URL, then paste the secret back here).
func (h *GatewayConnectionHandler) SetWebhookSecret(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok || !h.requireManager(c) {
		return
	}
	var req webhookSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if err := h.svc.SetWebhookSecret(c.Request.Context(), tenantID, c.Param("provider"), req.WebhookSecret); err != nil {
		h.respondConnErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// Disconnect soft-deletes the tenant's active connection for a provider.
func (h *GatewayConnectionHandler) Disconnect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok || !h.requireManager(c) {
		return
	}
	if err := h.svc.Disconnect(c.Request.Context(), tenantID, c.Param("provider")); err != nil {
		h.respondConnErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

// respondConnErr maps service errors onto HTTP status codes.
func (h *GatewayConnectionHandler) respondConnErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrGatewayConnectionNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, "no active connection for that provider")
	case errors.Is(err, domain.ErrGatewayVaultUnavailable):
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "credential vault unavailable (GATEWAY_ENCRYPTION_KEY not configured)")
	case service.IsGatewayConnectionValidationError(err):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	default:
		respondInternalError(c, err)
	}
}
