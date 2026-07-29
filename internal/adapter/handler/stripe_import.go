package handler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
)

// maxImportBytes caps an uploaded Stripe export so a huge (or hostile) body
// can't exhaust memory. 25 MB comfortably holds tens of thousands of objects.
const maxImportBytes = 25 << 20

// customerLister / planLister are the narrow slices of the catalog + customer
// services the importer needs to detect conflicts (existing emails / plan
// codes). *service.CustomerService and *service.CatalogService satisfy them;
// tests supply fakes.
type customerLister interface {
	ListCustomers(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
}
type planLister interface {
	ListPlans(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error)
}

// StripeImportHandler serves the Stripe migration endpoints. This first
// increment is preview-only (a dry run): it never writes. The commit endpoint
// (with idempotent external-ref persistence) lands in a follow-up.
type StripeImportHandler struct {
	customers customerLister
	plans     planLister
}

func NewStripeImportHandler(customers customerLister, plans planLister) *StripeImportHandler {
	return &StripeImportHandler{customers: customers, plans: plans}
}

// existingScanLimit bounds how many existing customers/plans we load to build
// the conflict sets. Large enough to cover real catalogs; a safety ceiling, not
// a business limit.
const existingScanLimit = 100000

// Preview parses an uploaded Stripe export and returns a dry-run plan of exactly
// what a commit would create, link, skip, or refuse — with zero side effects.
//
// POST /v1/import/stripe/preview
// Body: the Stripe export JSON ({"customers":[...],"products":[...],...}).
func (h *StripeImportHandler) Preview(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "missing tenant")
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxImportBytes+1))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "could not read request body")
		return
	}
	if len(body) > maxImportBytes {
		respondError(c, http.StatusRequestEntityTooLarge, codeValidationFailed, "export exceeds the 25 MB limit")
		return
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "empty request body; upload a Stripe export")
		return
	}

	exp, err := stripeimport.Parse(body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	existing, err := h.buildExisting(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	plan := stripeimport.BuildPlan(exp, existing)
	c.JSON(http.StatusOK, plan)
}

// buildExisting loads the tenant's current customer emails and plan codes so the
// planner can link (not duplicate) matches. ImportedStripeIDs is left empty here
// — full idempotency arrives with the external-ref table in the commit step.
func (h *StripeImportHandler) buildExisting(ctx context.Context, tenantID uuid.UUID) (stripeimport.Existing, error) {
	existing := stripeimport.Existing{
		CustomerEmails: map[string]bool{},
		PlanCodes:      map[string]bool{},
	}

	customers, err := h.customers.ListCustomers(ctx, tenantID, domain.CustomerFilter{Limit: existingScanLimit})
	if err != nil {
		return existing, err
	}
	for _, cust := range customers {
		if cust.Email != "" {
			existing.CustomerEmails[strings.ToLower(strings.TrimSpace(cust.Email))] = true
		}
	}

	plans, err := h.plans.ListPlans(ctx, tenantID, domain.PlanFilter{Limit: existingScanLimit})
	if err != nil {
		return existing, err
	}
	for _, pl := range plans {
		if pl.Code != "" {
			existing.PlanCodes[pl.Code] = true
		}
	}
	return existing, nil
}
