package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// collectionsQueueLister is the read-only slice of the invoice repository the
// collections worklist needs. *db.InvoiceRepository satisfies it.
type collectionsQueueLister interface {
	ListCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) ([]domain.CollectionsQueueItem, error)
	CountCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) (int, error)
}

// collectionsAnalytics is the recovery-funnel + failure-breakdown analytics the
// dashboard needs (Inc 2). *service.DunningRecoveryService satisfies it.
type collectionsAnalytics interface {
	GetCollectionsFunnel(ctx context.Context, tenantID uuid.UUID) (*service.CollectionsFunnel, error)
	GetFailureBreakdown(ctx context.Context, tenantID uuid.UUID) ([]service.CollectionsFailureBucket, error)
}

// collectionsActions is the operator-initiated mutation surface (Inc 3).
// *service.CollectionsActionService satisfies it.
type collectionsActions interface {
	RetryNow(ctx context.Context, tenantID, invoiceID uuid.UUID) error
	SetPaused(ctx context.Context, tenantID, invoiceID uuid.UUID, paused bool) error
	MarkUncollectible(ctx context.Context, tenantID, invoiceID uuid.UUID) error
}

// CollectionsHandler serves the operator-facing collections views + actions
// (Collections Intelligence). The read paths are read-only; the Inc 3 actions
// mutate a single invoice's dunning state (never the ledger).
type CollectionsHandler struct {
	repo      collectionsQueueLister
	analytics collectionsAnalytics // nil-safe
	actions   collectionsActions   // nil-safe
}

func NewCollectionsHandler(repo collectionsQueueLister, analytics collectionsAnalytics, actions collectionsActions) *CollectionsHandler {
	return &CollectionsHandler{repo: repo, analytics: analytics, actions: actions}
}

// validCollectionsStatus / validManagedBy guard the filter inputs so an
// arbitrary query string can't reach the SQL as an unexpected value (it would
// simply match nothing, but validating keeps the contract explicit).
var validCollectionsStatus = map[string]bool{"past_due": true, "uncollectible": true}
var validManagedBy = map[string]bool{"scheduler": true, "worker": true, "campaign": true}

// GetQueue returns the paginated collections worklist for the tenant:
// currently-failing invoices with their recovery state, customer, and latest
// ACH attempt status. GET /v1/collections/queue?status=&managed_by=&page=&per_page=
func (h *CollectionsHandler) GetQueue(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	p := ParsePagination(c)
	f := domain.CollectionsQueueFilter{Limit: p.Limit, Offset: p.Offset}
	if s := c.Query("status"); s != "" && validCollectionsStatus[s] {
		f.Status = s
	}
	if m := c.Query("managed_by"); m != "" && validManagedBy[m] {
		f.ManagedBy = m
	}

	items, err := h.repo.ListCollectionsQueue(c.Request.Context(), tenantID, f)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	total, err := h.repo.CountCollectionsQueue(c.Request.Context(), tenantID, f)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if items == nil {
		items = []domain.CollectionsQueueItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"page": p.Page, "per_page": p.PerPage, "total": total},
	})
}

// GetFunnel returns the recovery funnel (past_due → resolved as recovered vs
// uncollectible) with revenue-at-risk, FX-normalized to the reporting currency.
// GET /v1/analytics/collections/funnel
func (h *CollectionsHandler) GetFunnel(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if h.analytics == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "collections analytics not configured")
		return
	}
	funnel, err := h.analytics.GetCollectionsFunnel(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": funnel})
}

// GetFailures returns the failure-reason breakdown ranked by money at risk.
// GET /v1/analytics/collections/failures
func (h *CollectionsHandler) GetFailures(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if h.analytics == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "collections analytics not configured")
		return
	}
	buckets, err := h.analytics.GetFailureBreakdown(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if buckets == nil {
		buckets = []service.CollectionsFailureBucket{}
	}
	c.JSON(http.StatusOK, gin.H{"data": buckets})
}

// actionContext validates the tenant + :id path param and confirms actions are
// wired, returning the ids and false if the request can't proceed (it has
// already written the error response).
func (h *CollectionsHandler) actionContext(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, uuid.Nil, false
	}
	if h.actions == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "collections actions not configured")
		return uuid.Nil, uuid.Nil, false
	}
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, invoiceID, true
}

// respondActionError maps a collections-action service error to an HTTP status:
// not-found → 404, any refused-precondition → 409 Conflict, else 500.
func respondActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCollectionInvoiceNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, service.ErrRetryNotPastDue),
		errors.Is(err, service.ErrRetryPaused),
		errors.Is(err, service.ErrRetryMandate),
		errors.Is(err, service.ErrRetryInFlight):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	default:
		respondInternalError(c, err)
	}
}

// RetryNow requeues a failing invoice for an immediate worker retry.
// POST /v1/collections/invoices/:id/retry-now
func (h *CollectionsHandler) RetryNow(c *gin.Context) {
	tenantID, invoiceID, ok := h.actionContext(c)
	if !ok {
		return
	}
	if err := h.actions.RetryNow(c.Request.Context(), tenantID, invoiceID); err != nil {
		respondActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "requeued"}})
}

// PauseDunning pauses or resumes automated dunning on an invoice.
// POST /v1/collections/invoices/:id/pause  body: {"paused": true|false}
func (h *CollectionsHandler) PauseDunning(c *gin.Context) {
	tenantID, invoiceID, ok := h.actionContext(c)
	if !ok {
		return
	}
	var body struct {
		Paused bool `json:"paused"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "body must be {\"paused\": bool}")
		return
	}
	if err := h.actions.SetPaused(c.Request.Context(), tenantID, invoiceID, body.Paused); err != nil {
		respondActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"dunning_paused": body.Paused}})
}

// MarkUncollectible is the operator-initiated write-off (status change only).
// POST /v1/collections/invoices/:id/mark-uncollectible
func (h *CollectionsHandler) MarkUncollectible(c *gin.Context) {
	tenantID, invoiceID, ok := h.actionContext(c)
	if !ok {
		return
	}
	if err := h.actions.MarkUncollectible(c.Request.Context(), tenantID, invoiceID); err != nil {
		respondActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "uncollectible"}})
}
