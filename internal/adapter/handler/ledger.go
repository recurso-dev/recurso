package handler

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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

// GetTrialBalance returns the caller's tenant trial balance: every account with
// posted debit/credit totals, its normal-side balance, an abnormal-sign flag,
// and the debits==credits invariant. Read-only.
func (h *LedgerHandler) GetTrialBalance(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	entityID, ok2 := entityIDParam(c)
	if !ok2 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity_id")
		return
	}

	// ?consolidated=true rolls every entity's accounts up by code into one
	// tenant-wide view (Multi-Entity Books); otherwise ?entity_id scopes to a
	// single entity, and the default lists every entity's accounts.
	var tb *domain.TrialBalance
	var err error
	if entityID == nil && c.Query("consolidated") == "true" {
		tb, err = h.service.GetConsolidatedTrialBalance(c.Request.Context(), tenantID.(uuid.UUID))
	} else {
		tb, err = h.service.GetTrialBalance(c.Request.Context(), tenantID.(uuid.UUID), entityID)
	}
	if err != nil {
		log.Printf("ledger GetTrialBalance error: %v", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to build trial balance")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tb})
}

// GetDeferredRollforward returns the movement of the caller's Deferred Revenue
// account for a calendar month (?month=&year=): opening, added, released, and
// closing. Read-only.
func (h *LedgerHandler) GetDeferredRollforward(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid or missing month (1-12)")
		return
	}
	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2000 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid or missing year")
		return
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	rf, err := h.service.GetDeferredRollforward(c.Request.Context(), tenantID.(uuid.UUID), start, end)
	if err != nil {
		log.Printf("ledger GetDeferredRollforward error: %v", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to build deferred rollforward")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rf})
}

// ExportGL streams the caller's tenant general ledger as CSV — every posted
// transaction with both account codes/names, amount, and provenance. Read-only.
func (h *LedgerHandler) ExportGL(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	entityID, ok2 := entityIDParam(c)
	if !ok2 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity_id")
		return
	}

	entries, err := h.service.GeneralLedger(c.Request.Context(), tenantID.(uuid.UUID), entityID)
	if err != nil {
		log.Printf("ledger ExportGL error: %v", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to export general ledger")
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=general-ledger.csv")

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"transaction_id", "timestamp", "code",
		"debit_account_code", "debit_account_name",
		"credit_account_code", "credit_account_name",
		"amount", "reference_id", "description",
	})
	for _, e := range entries {
		_ = w.Write([]string{
			e.TransactionID.String(),
			e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			strconv.Itoa(int(e.Code)),
			strconv.Itoa(e.DebitAccountCode), e.DebitAccountName,
			strconv.Itoa(e.CreditAccountCode), e.CreditAccountName,
			fmt.Sprintf("%d", e.Amount),
			e.ReferenceID.String(),
			e.Description,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Printf("ledger ExportGL csv flush error: %v", err)
	}
}

func (h *LedgerHandler) ListAccounts(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		log.Printf("ledger ListAccounts error: %v", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch accounts")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}
