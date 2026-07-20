package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type OrganizationHandler struct {
	service *service.OrganizationService
}

func NewOrganizationHandler(s *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: s}
}

// callerTenantID pulls the authenticated tenant set by the dual-auth middleware.
func (h *OrganizationHandler) callerTenantID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := c.MustGet("tenant_id").(uuid.UUID)
	return id, ok
}

// mapOrgError translates organization-service errors to HTTP responses. A
// cross-tenant access is reported as 404 (no existence oracle); an attempt to
// attach a foreign tenant is a 403.
func mapOrgError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOrganizationNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, "organization not found")
	case errors.Is(err, domain.ErrCrossTenantAttach):
		respondError(c, http.StatusForbidden, codeForbidden, err.Error())
	default:
		respondInternalError(c, err)
	}
}

type createOrgRequest struct {
	Name       string `json:"name" binding:"required"`
	OwnerEmail string `json:"owner_email" binding:"required,email"`
}

func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	org, err := h.service.Create(c.Request.Context(), tenantID, req.Name, req.OwnerEmail)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, org)
}

func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	org, err := h.service.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": org})
}

type addTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

func (h *OrganizationHandler) AddTenant(c *gin.Context) {
	callerTenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	var req addTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid tenant_id")
		return
	}

	if err := h.service.AddTenant(c.Request.Context(), callerTenantID, orgID, tenantID); err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "added"})
}

func (h *OrganizationHandler) ListTenants(c *gin.Context) {
	callerTenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	tenants, err := h.service.ListTenants(c.Request.Context(), callerTenantID, orgID)
	if err != nil {
		mapOrgError(c, err)
		return
	}

	if tenants == nil {
		tenants = []*domain.Tenant{}
	}

	c.JSON(http.StatusOK, gin.H{"data": tenants})
}

type updateOrgRequest struct {
	Name       string `json:"name"`
	OwnerEmail string `json:"owner_email"`
}

func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	var req updateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	org, err := h.service.Update(c.Request.Context(), tenantID, id, req.Name, req.OwnerEmail)
	if err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": org})
}

func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), tenantID, id); err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *OrganizationHandler) RemoveTenant(c *gin.Context) {
	callerTenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid tenant id")
		return
	}

	if err := h.service.RemoveTenant(c.Request.Context(), callerTenantID, orgID, tenantID); err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	orgs, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if orgs == nil {
		orgs = []*domain.Organization{}
	}

	c.JSON(http.StatusOK, gin.H{"data": orgs})
}

func (h *OrganizationHandler) GetConsolidatedMRR(c *gin.Context) {
	tenantID, ok := h.callerTenantID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid organization id")
		return
	}

	metrics, err := h.service.GetConsolidatedMRR(c.Request.Context(), tenantID, orgID)
	if err != nil {
		mapOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}
