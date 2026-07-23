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

type CreditNoteHandler struct {
	service *service.CreditNoteService
}

func NewCreditNoteHandler(service *service.CreditNoteService) *CreditNoteHandler {
	return &CreditNoteHandler{service: service}
}

// GetCreditStatement returns a customer's consolidated account-credit statement:
// spendable balance (per currency/entity), grants, draw-down history, and a
// per-currency rollup.
// GET /v1/customers/:id/credit-statement
func (h *CreditNoteHandler) GetCreditStatement(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	stmt, err := h.service.GetCreditStatement(ctx, tenantID, customerID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stmt})
}

func (h *CreditNoteHandler) CreateCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var req domain.CreateCreditNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	cn, err := h.service.Create(ctx, tenantID, userID, userRole, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": cn})
}

func (h *CreditNoteHandler) ListCreditNotes(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var filter domain.CreditNoteFilter
	if customerIDStr := c.Query("customer_id"); customerIDStr != "" {
		if id, err := uuid.Parse(customerIDStr); err == nil {
			filter.CustomerID = &id
		}
	}

	// Status filter logic can be added later

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cns, err := h.service.List(ctx, tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cns})
}

func (h *CreditNoteHandler) ApproveCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cnIDStr := c.Param("id")
	cnID, err := uuid.Parse(cnIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	if userRole != "" && userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleOwner) {
		respondError(c, http.StatusForbidden, codeValidationFailed, "only admins and owners can approve credit notes")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Approve(ctx, tenantID, cnID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cn})
}

func (h *CreditNoteHandler) RejectCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cnIDStr := c.Param("id")
	cnID, err := uuid.Parse(cnIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	if userRole != "" && userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleOwner) {
		respondError(c, http.StatusForbidden, codeValidationFailed, "only admins and owners can reject credit notes")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Reject(ctx, tenantID, cnID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cn})
}
