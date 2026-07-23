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
	GetByTenantEntity(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) (*domain.TenantEUConfig, error)
	Upsert(ctx context.Context, entityID *uuid.UUID, c *domain.TenantEUConfig) error
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
	entityID, ok := entityIDParam(c)
	if !ok {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity_id")
		return
	}
	cfg, err := h.repo.GetByTenantEntity(c.Request.Context(), tenantID, entityID)
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
	entityID, ok := entityIDParam(c)
	if !ok {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity_id")
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
	if err := h.repo.Upsert(c.Request.Context(), entityID, cfg); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": in})
}

// --- Per-invoice EU e-invoice inspection + manual retry (Track C inc 2) ---

// euInvoiceReader / euInvoiceOwnerLookup / euCustomerLookup / euGenerator are the
// narrow deps the per-invoice handler needs, satisfied by *db.EUInvoiceRepository,
// *db.InvoiceRepository, *db.CustomerRepository and *service.EUEInvoiceService.
type euInvoiceReader interface {
	GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.EUInvoice, error)
}
type euInvoiceOwnerLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
}
type euCustomerLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
}
type euGenerator interface {
	GenerateForInvoice(ctx context.Context, inv *domain.Invoice, customer *domain.Customer) (*domain.EUInvoice, error)
}

// EUEInvoiceHandler serves an individual invoice's EU e-invoice: its generated
// EN 16931 UBL document and delivery status, plus a manual regenerate/re-transmit
// for a failed one. Ownership is checked by loading the invoice and matching the
// tenant (the eu_einvoices read is not tenant-scoped on its own).
type EUEInvoiceHandler struct {
	euInvoices euInvoiceReader
	invoices   euInvoiceOwnerLookup
	customers  euCustomerLookup
	svc        euGenerator
}

func NewEUEInvoiceHandler(euInvoices euInvoiceReader, invoices euInvoiceOwnerLookup, customers euCustomerLookup, svc euGenerator) *EUEInvoiceHandler {
	return &EUEInvoiceHandler{euInvoices: euInvoices, invoices: invoices, customers: customers, svc: svc}
}

// ownedInvoice loads the invoice and confirms it belongs to the caller's tenant,
// writing the appropriate error response and returning ok=false otherwise.
func (h *EUEInvoiceHandler) ownedInvoice(c *gin.Context) (*domain.Invoice, uuid.UUID, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return nil, uuid.Nil, false
	}
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return nil, uuid.Nil, false
	}
	inv, err := h.invoices.GetByID(c.Request.Context(), invoiceID)
	if err != nil {
		respondInternalError(c, err)
		return nil, uuid.Nil, false
	}
	if inv == nil || inv.TenantID != tenantID {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return nil, uuid.Nil, false
	}
	return inv, tenantID, true
}

// GetEUEInvoice returns the invoice's EU e-invoice record (status, syntax, UBL
// document, delivery id / error), or data:null when none has been generated.
// GET /v1/invoices/:id/eu-einvoice
func (h *EUEInvoiceHandler) GetEUEInvoice(c *gin.Context) {
	inv, _, ok := h.ownedInvoice(c)
	if !ok {
		return
	}
	rec, err := h.euInvoices.GetByInvoiceID(c.Request.Context(), inv.ID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec}) // rec == nil ⇒ data:null (not generated)
}

// RetryEUEInvoice regenerates and re-transmits the invoice's EU e-invoice. It
// re-runs the same generation the post-commit hook uses, so it recovers both a
// generation failure (data since corrected) and a transmission failure, and is
// idempotent (upsert on invoice_id). data:null means the tenant hasn't opted in.
// POST /v1/invoices/:id/eu-einvoice/retry
func (h *EUEInvoiceHandler) RetryEUEInvoice(c *gin.Context) {
	inv, _, ok := h.ownedInvoice(c)
	if !ok {
		return
	}
	customer, err := h.customers.GetByID(c.Request.Context(), inv.CustomerID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	// GenerateForInvoice records a failure on the stored record and returns it
	// alongside the error, so surface the (updated) record either way — the
	// client reads status/error_message rather than relying on the HTTP code.
	rec, genErr := h.svc.GenerateForInvoice(c.Request.Context(), inv, customer)
	msg := "EU e-invoice regenerated"
	if rec == nil && genErr == nil {
		msg = "EU e-invoicing is not enabled for this tenant"
	} else if genErr != nil {
		msg = "retry ran but the EU e-invoice is still failing"
	}
	c.JSON(http.StatusOK, gin.H{"data": rec, "message": msg})
}
