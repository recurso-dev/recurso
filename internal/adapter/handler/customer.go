package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// parseExemptionExpiry parses a YYYY-MM-DD certificate expiry into a *time.Time.
// Empty ⇒ nil (no expiry on file).
func parseExemptionExpiry(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("tax_exemption_expires_at must be a date (YYYY-MM-DD)")
	}
	return &t, nil
}

type CustomerHandler struct {
	service *service.CustomerService
	subs    port.SubscriptionRepository
}

func NewCustomerHandler(s *service.CustomerService, subs port.SubscriptionRepository) *CustomerHandler {
	return &CustomerHandler{service: s, subs: subs}
}

// billingAddress is the nested address shape some clients send instead of the
// flat line1/city/state/zip/country fields. Accepted as an alias so a nested
// object is no longer silently dropped (verification finding S2). Flat fields
// win when both are present; postal_code and zip are synonyms.
type billingAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        string `json:"zip"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// coalesce returns the first non-empty argument.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type createCustomerRequest struct {
	Email          string          `json:"email" binding:"required,email"`
	Name           string          `json:"name" binding:"required"`
	Phone          string          `json:"phone"`
	TaxID          string          `json:"tax_id"`
	GSTIN          string          `json:"gstin"`           // P24
	TaxType        string          `json:"tax_type"`        // P25
	PlaceOfSupply  string          `json:"place_of_supply"` // P24
	Line1          string          `json:"line1"`
	City           string          `json:"city"`
	State          string          `json:"state"`
	Zip            string          `json:"zip"`
	Country        string          `json:"country" binding:"omitempty,len=2"` // Allow empty or iso code
	BillingAddress *billingAddress `json:"billing_address"`                   // nested alias for the flat fields
	// US sales-tax exemption (D2)
	TaxExempt             bool   `json:"tax_exempt"`
	TaxExemptionNumber    string `json:"tax_exemption_number"`
	TaxExemptionCode      string `json:"tax_exemption_code"`
	TaxExemptionExpiresAt string `json:"tax_exemption_expires_at"` // YYYY-MM-DD; empty = no expiry (Inc 2)
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req createCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	expiry, err := parseExemptionExpiry(req.TaxExemptionExpiresAt)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	// Fold a nested billing_address into the flat fields (flat wins).
	if a := req.BillingAddress; a != nil {
		req.Line1 = coalesce(req.Line1, a.Line1)
		req.City = coalesce(req.City, a.City)
		req.State = coalesce(req.State, a.State)
		req.Zip = coalesce(req.Zip, a.Zip, a.PostalCode)
		req.Country = coalesce(req.Country, a.Country)
	}

	input := service.CreateCustomerInput{
		TenantID:      tenantID,
		Email:         req.Email,
		Name:          req.Name,
		Phone:         req.Phone,
		TaxID:         req.TaxID,
		GSTIN:         req.GSTIN,
		TaxType:       req.TaxType,
		PlaceOfSupply: req.PlaceOfSupply,
		Line1:         req.Line1,
		City:          req.City,
		State:         req.State,
		Zip:           req.Zip,
		Country:       req.Country,

		TaxExempt:             req.TaxExempt,
		TaxExemptionNumber:    req.TaxExemptionNumber,
		TaxExemptionCode:      req.TaxExemptionCode,
		TaxExemptionExpiresAt: expiry,
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	customer, err := h.service.CreateCustomer(ctx, input)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

type updatePaymentMethodRequest struct {
	CardBrand string `json:"card_brand" binding:"required"`
	CardLast4 string `json:"card_last4" binding:"required,len=4"`
	ExpMonth  int    `json:"card_exp_month" binding:"required,min=1,max=12"`
	ExpYear   int    `json:"card_exp_year" binding:"required,min=2020"`
}

func (h *CustomerHandler) UpdatePaymentMethod(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}

	var req updatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	input := service.UpdatePaymentMethodInput{
		CustomerID: customerID,
		CardBrand:  req.CardBrand,
		CardLast4:  req.CardLast4,
		ExpMonth:   req.ExpMonth,
		ExpYear:    req.ExpYear,
	}

	// Inject the tenant so the repo scopes the UPDATE — otherwise any tenant
	// could overwrite another tenant's customer's card (cross-tenant IDOR).
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	if err := h.service.UpdatePaymentMethod(ctx, input); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "customer not found")
			return
		}
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetCustomer handles GET /v1/customers/:id.
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	customer, err := h.service.GetCustomer(ctx, tenantID, customerID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if customer == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "customer not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": customer})
}

// updateCustomerRequest is a partial update: nil fields are left unchanged, so
// the same endpoint edits contact/tax details or archives (active=false).
type updateCustomerRequest struct {
	Name           *string         `json:"name"`
	Email          *string         `json:"email" binding:"omitempty,email"`
	Phone          *string         `json:"phone"`
	TaxID          *string         `json:"tax_id"`
	GSTIN          *string         `json:"gstin"`
	TaxType        *string         `json:"tax_type" binding:"omitempty,oneof=business consumer"`
	PlaceOfSupply  *string         `json:"place_of_supply"`
	Line1          *string         `json:"line1"`
	City           *string         `json:"city"`
	State          *string         `json:"state"`
	Zip            *string         `json:"zip"`
	Country        *string         `json:"country" binding:"omitempty,len=2"`
	BillingAddress *billingAddress `json:"billing_address"` // nested alias for the flat fields
	Active         *bool           `json:"active"`
	// US sales-tax exemption (D2)
	TaxExempt             *bool   `json:"tax_exempt"`
	TaxExemptionNumber    *string `json:"tax_exemption_number"`
	TaxExemptionCode      *string `json:"tax_exemption_code"`
	TaxExemptionExpiresAt *string `json:"tax_exemption_expires_at"` // nil=unchanged; ""=clear; YYYY-MM-DD=set (Inc 2)
}

// UpdateCustomer handles PUT /v1/customers/:id. Archiving (active=false) is
// refused while the customer has active subscriptions.
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}

	var req updateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	// Fold a nested billing_address into the flat pointer fields, but only
	// where the caller didn't set the flat field (flat wins). A nested field
	// only overlays when non-empty, so this never clears an unset flat field.
	if a := req.BillingAddress; a != nil {
		if req.Line1 == nil && a.Line1 != "" {
			req.Line1 = &a.Line1
		}
		if req.City == nil && a.City != "" {
			req.City = &a.City
		}
		if req.State == nil && a.State != "" {
			req.State = &a.State
		}
		if req.Zip == nil {
			if z := coalesce(a.Zip, a.PostalCode); z != "" {
				req.Zip = &z
			}
		}
		if req.Country == nil && a.Country != "" {
			req.Country = &a.Country
		}
	}

	// Partial: only touch the exemption expiry when the field is present.
	var expiry *time.Time
	expirySet := false
	if req.TaxExemptionExpiresAt != nil {
		parsed, perr := parseExemptionExpiry(*req.TaxExemptionExpiresAt)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, perr.Error())
			return
		}
		expiry = parsed
		expirySet = true
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	customer, err := h.service.UpdateCustomer(ctx, service.UpdateCustomerInput{
		TenantID:      tenantID,
		CustomerID:    customerID,
		Name:          req.Name,
		Email:         req.Email,
		Phone:         req.Phone,
		TaxID:         req.TaxID,
		GSTIN:         req.GSTIN,
		TaxType:       req.TaxType,
		PlaceOfSupply: req.PlaceOfSupply,
		Line1:         req.Line1,
		City:          req.City,
		State:         req.State,
		Zip:           req.Zip,
		Country:       req.Country,
		Active:        req.Active,

		TaxExempt:                req.TaxExempt,
		TaxExemptionNumber:       req.TaxExemptionNumber,
		TaxExemptionCode:         req.TaxExemptionCode,
		TaxExemptionExpiresAt:    expiry,
		TaxExemptionExpiresAtSet: expirySet,
	})
	if err != nil {
		// The archive gate ("active subscriptions") and field validation are
		// caller-fixable, so they surface as 400s.
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if customer == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "customer not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	// Parse query params
	search := c.Query("q")
	country := c.Query("country")
	status := c.Query("status")

	limit, offset := parsePageLimit(c)

	filter := domain.CustomerFilter{
		Search:  search,
		Country: country,
		Status:  status,
		Limit:   limit,
		Offset:  offset,
	}

	customers, err := h.service.ListCustomers(ctx, tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if customers == nil {
		customers = []*domain.Customer{}
	}

	// Attach each customer's active-subscription count so the list can show a
	// real Active/Inactive status and sub count. The count is best-effort — a
	// failure here shouldn't fail the whole list (customers just show 0).
	counts, err := h.subs.CountActiveByCustomer(ctx, tenantID)
	if err != nil {
		counts = map[uuid.UUID]int{}
	}
	type customerWithSubs struct {
		*domain.Customer
		ActiveSubs int `json:"active_subs"`
	}
	out := make([]customerWithSubs, len(customers))
	for i, cust := range customers {
		out[i] = customerWithSubs{Customer: cust, ActiveSubs: counts[cust.ID]}
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}
