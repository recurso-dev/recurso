package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// entityService is the surface the handler needs; *service.EntityService satisfies it.
type entityService interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Entity, error)
	Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.Entity, error)
	Create(ctx context.Context, tenantID uuid.UUID, in service.CreateEntityInput) (*domain.Entity, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, in service.CreateEntityInput) (*domain.Entity, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// EntityHandler exposes /v1/entities — a tenant's legal entities.
type EntityHandler struct {
	svc entityService
}

func NewEntityHandler(svc entityService) *EntityHandler {
	return &EntityHandler{svc: svc}
}

type entityRequest struct {
	Name          string `json:"name"`
	LegalName     string `json:"legal_name"`
	InvoicePrefix string `json:"invoice_prefix"`
	CountryCode   string `json:"country_code"`
}

func (r entityRequest) toInput() service.CreateEntityInput {
	return service.CreateEntityInput{
		Name:          r.Name,
		LegalName:     r.LegalName,
		InvoicePrefix: r.InvoicePrefix,
		CountryCode:   r.CountryCode,
	}
}

func (h *EntityHandler) entityError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, service.ErrEntityValidation) {
		status = http.StatusBadRequest
	}
	if status == http.StatusInternalServerError {
		respondInternalError(c, err)
		return
	}
	respondErrorStatus(c, status, err.Error())
}

func (h *EntityHandler) ListEntities(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	entities, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if entities == nil {
		entities = []*domain.Entity{}
	}
	c.JSON(http.StatusOK, gin.H{"data": entities})
}

func (h *EntityHandler) GetEntity(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity id")
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		h.entityError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}

func (h *EntityHandler) CreateEntity(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var req entityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	e, err := h.svc.Create(c.Request.Context(), tenantID, req.toInput())
	if err != nil {
		h.entityError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": e})
}

func (h *EntityHandler) UpdateEntity(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity id")
		return
	}
	var req entityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	e, err := h.svc.Update(c.Request.Context(), tenantID, id, req.toInput())
	if err != nil {
		h.entityError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}

func (h *EntityHandler) DeleteEntity(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), tenantID, id); err != nil {
		h.entityError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
