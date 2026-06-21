package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/middleware"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cn, err := h.service.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	cns, err := h.service.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cns})
}
