package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type TenantHandler struct {
	service *service.TenantService
}

// requireManager gates API-key management to owner/admin. An API-key (machine)
// caller carries no user role and is trusted (it already has full tenant
// access); a dashboard user must be owner or admin. Without this any member
// could mint a live-mode key granting full tenant API access (ENG-178).
func (h *TenantHandler) requireManager(c *gin.Context) bool {
	role, hasUser := middleware.GetUserRole(c)
	if !hasUser {
		return true // machine (API key) caller
	}
	if domain.Role(role).CanManageTeam() {
		return true
	}
	respondError(c, http.StatusForbidden, codeForbidden, "requires owner or admin role")
	return false
}

func NewTenantHandler(service *service.TenantService) *TenantHandler {
	return &TenantHandler{service: service}
}

type RegisterRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *TenantHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenant, apiKey, err := h.service.Register(c.Request.Context(), req.Name, req.Email)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tenant":  tenant,
		"api_key": apiKey.KeyValue,
	})
}

func (h *TenantHandler) ListKeys(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}

	keys, err := h.service.ListKeys(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if keys == nil {
		keys = []*domain.APIKey{} // Avoid null in JSON
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func (h *TenantHandler) CreateKey(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}

	// Optional body: {"name": "...", "mode": "test"|"live"}. Defaults to a
	// test-mode key — the safe default, and the one that works on a dev server.
	var req struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
	}
	_ = c.ShouldBindJSON(&req)
	name := req.Name
	if name == "" {
		name = "New Key"
	}
	livemode := strings.EqualFold(req.Mode, "live")

	key, err := h.service.GenerateKey(c.Request.Context(), tenantID, name, livemode)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, key)
}

// RevokeKey handles DELETE /v1/developer/keys/:id — soft-deactivates the key
// (auth filters on is_active, so it stops working immediately). Manager-only,
// like key creation.
func (h *TenantHandler) RevokeKey(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid key id")
		return
	}

	if err := h.service.RevokeKey(c.Request.Context(), tenantID, keyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "key not found or already revoked")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

func (h *TenantHandler) GetAccount(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	account, err := h.service.GetAccount(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}

type UpdateAccountRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *TenantHandler) UpdateAccount(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	account, err := h.service.UpdateAccount(c.Request.Context(), tenantID, req.Name, req.Email)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}
