package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// QuoteHandler handles quote API endpoints
type QuoteHandler struct {
	quoteService *service.QuoteService
}

func NewQuoteHandler(quoteService *service.QuoteService) *QuoteHandler {
	return &QuoteHandler{quoteService: quoteService}
}

// CreateQuote handles POST /quotes
func (h *QuoteHandler) CreateQuote(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	var req domain.CreateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	quote, err := h.quoteService.CreateQuote(c.Request.Context(), tenantID, req)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": quote})
}

// GetQuote handles GET /quotes/:id
func (h *QuoteHandler) GetQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	quote, err := h.quoteService.GetQuote(c.Request.Context(), id, tenantID)
	if err != nil {
		respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quote})
}

// ListQuotes handles GET /quotes
func (h *QuoteHandler) ListQuotes(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	filter := domain.QuoteFilter{
		Status:     c.Query("status"),
		CustomerID: c.Query("customer_id"),
		Search:     c.Query("search"),
	}

	quotes, err := h.quoteService.ListQuotes(c.Request.Context(), tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quotes})
}

// UpdateQuote handles PUT /quotes/:id
func (h *QuoteHandler) UpdateQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	var req domain.CreateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	quote, err := h.quoteService.UpdateQuote(c.Request.Context(), id, tenantID, req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		if err == service.ErrQuoteNotEditable {
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
			return
		}
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quote})
}

// DeleteQuote handles DELETE /quotes/:id
func (h *QuoteHandler) DeleteQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	if err := h.quoteService.DeleteQuote(c.Request.Context(), id, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		if err == service.ErrQuoteNotEditable {
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
			return
		}
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quote deleted"})
}

// SendQuote handles POST /quotes/:id/send
func (h *QuoteHandler) SendQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	quote, err := h.quoteService.SendQuote(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quote, "message": "quote sent"})
}

// AcceptQuote handles POST /quotes/:id/accept
func (h *QuoteHandler) AcceptQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	quote, err := h.quoteService.AcceptQuote(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quote, "message": "quote accepted"})
}

// DeclineQuote handles POST /quotes/:id/decline
func (h *QuoteHandler) DeclineQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	quote, err := h.quoteService.DeclineQuote(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quote, "message": "quote declined"})
}

// ConvertToInvoice handles POST /quotes/:id/convert
func (h *QuoteHandler) ConvertToInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid quote id")
		return
	}

	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	invoice, err := h.quoteService.ConvertToInvoice(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "quote not found")
			return
		}
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": invoice, "message": "invoice created from quote"})
}
