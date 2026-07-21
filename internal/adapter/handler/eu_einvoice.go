package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// euConfigStore is the persistence the handler needs; satisfied by
// *db.TenantEUConfigRepository.
type euConfigStore interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantEUConfig, error)
	Upsert(ctx context.Context, c *domain.TenantEUConfig) error
}

// EUConfigHandler manages a tenant's EU e-invoicing configuration (Track C):
// the opt-in flag plus the EN 16931 seller party. Kept separate from the India
// GST settings so the regional compliance boundaries stay clean.
type EUConfigHandler struct {
	repo euConfigStore
}

func NewEUConfigHandler(repo euConfigStore) *EUConfigHandler {
	return &EUConfigHandler{repo: repo}
}

// EUConfigDTO is the request/response shape for the EU e-invoicing settings.
type EUConfigDTO struct {
	Enabled     bool   `json:"enabled"`
	LegalName   string `json:"legal_name"`
	VATNumber   string `json:"vat_number"`
	CountryCode string `json:"country_code"`
	Street      string `json:"street"`
	City        string `json:"city"`
	PostalZone  string `json:"postal_zone"`
}

// GetEUConfig returns the tenant's EU e-invoicing config, or an empty (disabled)
// default when none is set yet.
func (h *EUConfigHandler) GetEUConfig(c *gin.Context) {
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
		c.JSON(http.StatusOK, gin.H{"data": EUConfigDTO{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": EUConfigDTO{
		Enabled:     cfg.Enabled,
		LegalName:   cfg.LegalName,
		VATNumber:   cfg.VATNumber,
		CountryCode: cfg.CountryCode,
		Street:      cfg.Street,
		City:        cfg.City,
		PostalZone:  cfg.PostalZone,
	}})
}

// UpdateEUConfig upserts the tenant's EU e-invoicing config. Enabling it requires
// a complete seller identity (name, VAT id, 2-letter country) — the fields every
// generated EN 16931 document needs — so a tenant can't opt in to a config that
// would fail on the first invoice.
func (h *EUConfigHandler) UpdateEUConfig(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	var in EUConfigDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	in.VATNumber = strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(in.VATNumber, " ", "")))

	if in.CountryCode != "" && len(in.CountryCode) != 2 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "country_code must be a 2-letter ISO code")
		return
	}
	if in.Enabled {
		switch {
		case strings.TrimSpace(in.LegalName) == "":
			respondError(c, http.StatusBadRequest, codeValidationFailed, "legal_name is required to enable EU e-invoicing")
			return
		case in.VATNumber == "":
			respondError(c, http.StatusBadRequest, codeValidationFailed, "vat_number is required to enable EU e-invoicing")
			return
		case len(in.CountryCode) != 2:
			respondError(c, http.StatusBadRequest, codeValidationFailed, "country_code is required to enable EU e-invoicing")
			return
		}
	}

	cfg := &domain.TenantEUConfig{
		TenantID:    tenantID,
		Enabled:     in.Enabled,
		LegalName:   strings.TrimSpace(in.LegalName),
		VATNumber:   in.VATNumber,
		CountryCode: in.CountryCode,
		Street:      strings.TrimSpace(in.Street),
		City:        strings.TrimSpace(in.City),
		PostalZone:  strings.TrimSpace(in.PostalZone),
	}
	if err := h.repo.Upsert(c.Request.Context(), cfg); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": in})
}
