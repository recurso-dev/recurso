package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/service"
)

type LedgerHandler struct {
	service *service.LedgerService
}

func NewLedgerHandler(service *service.LedgerService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

func (h *LedgerHandler) GetEntries(c *gin.Context) {
	accountIDStr := c.Query("account_id")
	if accountIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	entries, err := h.service.GetEntries(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ledger entries"})
		return
	}

	if entries == nil {
		entries = []*domain.LedgerTransaction{}
	}

	c.JSON(http.StatusOK, gin.H{"data": entries})
}

func (h *LedgerHandler) ListAccounts(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}
