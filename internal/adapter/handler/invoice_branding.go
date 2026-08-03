package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// invoiceBrandingStore is the persistence the handler needs; satisfied by
// *db.InvoiceBrandingRepository.
type invoiceBrandingStore interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantInvoiceBranding, error)
	Upsert(ctx context.Context, b *domain.TenantInvoiceBranding) error
}

// InvoiceBrandingHandler manages a tenant's invoice presentation settings:
// display name, logo, signature, signatory, bank details and terms. This is
// the "make the invoice look like ours" surface — statutory identity stays in
// the GST / W-9 settings.
type InvoiceBrandingHandler struct {
	repo invoiceBrandingStore
}

func NewInvoiceBrandingHandler(repo invoiceBrandingStore) *InvoiceBrandingHandler {
	return &InvoiceBrandingHandler{repo: repo}
}

// InvoiceBrandingDTO is the request/response shape for the branding settings.
type InvoiceBrandingDTO struct {
	CompanyName      string `json:"company_name"`
	LogoDataURL      string `json:"logo_data_url"`
	SignatureDataURL string `json:"signature_data_url"`
	SignatoryName    string `json:"signatory_name"`
	BankDetails      string `json:"bank_details"`
	Terms            string `json:"terms"`
}

// Image data URLs are validated to this exact shape before storage. The strict
// character class (base64 alphabet only — no quotes, angle brackets or spaces)
// is what makes it safe to render the stored value as template.URL in the
// invoice HTML: nothing that survives this regex can escape an attribute.
// SVG is deliberately excluded (script-capable).
var imageDataURLRe = regexp.MustCompile(`^data:image/(png|jpeg);base64,[A-Za-z0-9+/]+=*$`)

const maxImageBytes = 300 * 1024 // 300KB decoded — plenty for a logo/signature

func validateImageDataURL(field, v string) error {
	if v == "" {
		return nil
	}
	if !imageDataURLRe.MatchString(v) {
		return fmt.Errorf("%s must be a data:image/png or data:image/jpeg base64 URL", field)
	}
	b64 := v[strings.Index(v, ",")+1:]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("%s is not valid base64", field)
	}
	if len(raw) > maxImageBytes {
		return fmt.Errorf("%s is too large (max %dKB)", field, maxImageBytes/1024)
	}
	return nil
}

// GetBranding returns the tenant's invoice branding, or an empty default when
// none is set yet.
// GET /v1/settings/invoice-branding
func (h *InvoiceBrandingHandler) GetBranding(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	b, err := h.repo.GetByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if b == nil {
		c.JSON(http.StatusOK, gin.H{"data": InvoiceBrandingDTO{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": InvoiceBrandingDTO{
		CompanyName:      b.CompanyName,
		LogoDataURL:      b.LogoDataURL,
		SignatureDataURL: b.SignatureDataURL,
		SignatoryName:    b.SignatoryName,
		BankDetails:      b.BankDetails,
		Terms:            b.Terms,
	}})
}

// UpdateBranding upserts the tenant's invoice branding.
// PUT /v1/settings/invoice-branding
func (h *InvoiceBrandingHandler) UpdateBranding(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	var in InvoiceBrandingDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	in.CompanyName = strings.TrimSpace(in.CompanyName)
	in.SignatoryName = strings.TrimSpace(in.SignatoryName)
	in.BankDetails = strings.TrimSpace(in.BankDetails)
	in.Terms = strings.TrimSpace(in.Terms)

	if len(in.CompanyName) > 200 || len(in.SignatoryName) > 200 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "name fields are limited to 200 characters")
		return
	}
	if len(in.BankDetails) > 4000 || len(in.Terms) > 4000 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "bank details and terms are limited to 4000 characters")
		return
	}
	if err := validateImageDataURL("logo", in.LogoDataURL); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if err := validateImageDataURL("signature", in.SignatureDataURL); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if err := h.repo.Upsert(c.Request.Context(), &domain.TenantInvoiceBranding{
		TenantID:         tenantID,
		CompanyName:      in.CompanyName,
		LogoDataURL:      in.LogoDataURL,
		SignatureDataURL: in.SignatureDataURL,
		SignatoryName:    in.SignatoryName,
		BankDetails:      in.BankDetails,
		Terms:            in.Terms,
	}); err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": in})
}
