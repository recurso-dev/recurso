package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

type LedgerHandler struct {
	service *service.LedgerService
}

func NewLedgerHandler(service *service.LedgerService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

func (h *LedgerHandler) GetEntries(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	accountIDStr := c.Query("account_id")
	if accountIDStr == "" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "account_id is required")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid account_id")
		return
	}

	entries, err := h.service.GetEntries(c.Request.Context(), tenantID.(uuid.UUID), accountID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch ledger entries")
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch accounts")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}
