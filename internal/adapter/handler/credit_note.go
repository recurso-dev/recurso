package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/middleware"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

type CreditNoteHandler struct {
	service *service.CreditNoteService
}

func NewCreditNoteHandler(service *service.CreditNoteService) *CreditNoteHandler {
	return &CreditNoteHandler{service: service}
}

func (h *CreditNoteHandler) CreateCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var req domain.CreateCreditNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Create(ctx, tenantID, req)
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cns})
}
