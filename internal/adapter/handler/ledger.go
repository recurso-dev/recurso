package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
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

	// Optional posting-code filter (?code=3) + standard limit/offset paging —
	// the page used to be silently capped at the repo's hard 100.
	var code uint16
	if cs := c.Query("code"); cs != "" {
		if v, err := strconv.Atoi(cs); err == nil && v > 0 && v < 65536 {
			code = uint16(v)
		}
	}
	pg := ParsePagination(c)
	entries, err := h.service.GetEntries(c.Request.Context(), tenantID.(uuid.UUID), accountID, code, pg.Limit, pg.Offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch ledger entries")
		return
	}

	if entries == nil {
		entries = []*domain.LedgerTransaction{}
	}

	c.JSON(http.StatusOK, gin.H{"data": entries})
}

// GetTransaction returns one posted transaction (a single journal entry) by its
// id — the addressable journal-entry object. Each leg carries its account id so
// the caller can deep-link to the account page. Cross-tenant ids return 404.
// Read-only.
func (h *LedgerHandler) GetTransaction(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	txID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid transaction id")
		return
	}
	row, err := h.service.GetTransactionByID(c.Request.Context(), tenantID.(uuid.UUID), txID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch transaction")
		return
	}
	if row == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "transaction not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": row})
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
		slog.Error("ledger GetTrialBalance error", "error", err)
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
		slog.Error("ledger GetDeferredRollforward error", "error", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to build deferred rollforward")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rf})
}

// ExportGL streams the caller's tenant general ledger as CSV — every posted
// transaction with both account codes/names, amount, and provenance. Read-only.
// Pass ?month=&year= (both together) to export one calendar month's postings —
// what the month-end close pack links to; omit both for the full ledger.
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

	var from, to *time.Time
	filename := "general-ledger.csv"
	if c.Query("month") != "" || c.Query("year") != "" {
		month, err := strconv.Atoi(c.Query("month"))
		if err != nil || month < 1 || month > 12 {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid month (1-12); pass month and year together")
			return
		}
		year, err := strconv.Atoi(c.Query("year"))
		if err != nil || year < 2000 {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid year; pass month and year together")
			return
		}
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		from, to = &start, &end
		filename = fmt.Sprintf("general-ledger-%04d-%02d.csv", year, month)
	}

	entries, err := h.service.GeneralLedger(c.Request.Context(), tenantID.(uuid.UUID), entityID, from, to)
	if err != nil {
		slog.Error("ledger ExportGL error", "error", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to export general ledger")
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	w := csv.NewWriter(c.Writer)
	// accounting_version is appended last so existing column positions are
	// unchanged for any downstream parser (ADR-008 journal provenance).
	_ = w.Write([]string{
		"transaction_id", "timestamp", "code",
		"debit_account_code", "debit_account_name",
		"credit_account_code", "credit_account_name",
		"amount", "reference_id", "description", "accounting_version",
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
			strconv.Itoa(e.AccountingVersion),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		slog.Error("ledger ExportGL csv flush error", "error", err)
	}
}

func (h *LedgerHandler) ListAccounts(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	limit, offset := parseLimitOffset(c, 1000, 1000)
	accounts, err := h.service.ListAccounts(c.Request.Context(), tenantID.(uuid.UUID), limit, offset)
	if err != nil {
		slog.Error("ledger ListAccounts error", "error", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch accounts")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}
