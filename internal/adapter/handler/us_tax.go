package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// usTaxConfigStore is the persistence the handler needs; satisfied by
// *db.TenantUSTaxConfigRepository.
type usTaxConfigStore interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUSTaxConfig, error)
	Upsert(ctx context.Context, c *domain.TenantUSTaxConfig) error
}

// USTaxConfigHandler manages a tenant's US tax identity (W-9): the seller party
// shown on US sales-tax invoices. Kept separate from the India GST and EU
// settings so the regional boundaries stay clean.
type USTaxConfigHandler struct {
	repo usTaxConfigStore
}

func NewUSTaxConfigHandler(repo usTaxConfigStore) *USTaxConfigHandler {
	return &USTaxConfigHandler{repo: repo}
}

// USTaxConfigDTO is the request/response shape for the US tax settings.
type USTaxConfigDTO struct {
	LegalName string `json:"legal_name"`
	EIN       string `json:"ein"`
	Address   string `json:"address"`
}

// GetUSTaxConfig returns the tenant's US tax config, or an empty default when
// none is set yet.
func (h *USTaxConfigHandler) GetUSTaxConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	cfg, err := h.repo.GetByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if cfg == nil {
		c.JSON(http.StatusOK, gin.H{"data": USTaxConfigDTO{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": USTaxConfigDTO{
		LegalName: cfg.LegalName,
		EIN:       cfg.EIN,
		Address:   cfg.Address,
	}})
}

// UpdateUSTaxConfig upserts the tenant's US tax config.
func (h *USTaxConfigHandler) UpdateUSTaxConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	var in USTaxConfigDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	in.LegalName = strings.TrimSpace(in.LegalName)
	in.EIN = strings.TrimSpace(in.EIN)
	in.Address = strings.TrimSpace(in.Address)

	if err := h.repo.Upsert(c.Request.Context(), &domain.TenantUSTaxConfig{
		TenantID:  tenantID,
		LegalName: in.LegalName,
		EIN:       in.EIN,
		Address:   in.Address,
	}); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": in})
}
