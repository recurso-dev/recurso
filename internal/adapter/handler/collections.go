package handler

import (
	"context"
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

// CollectionsHandler serves the operator-facing collections views
// (Collections Intelligence). Read-only; the automated recovery engine is
// untouched.
type CollectionsHandler struct {
	repo      collectionsQueueLister
	analytics collectionsAnalytics // nil-safe
}

func NewCollectionsHandler(repo collectionsQueueLister, analytics collectionsAnalytics) *CollectionsHandler {
	return &CollectionsHandler{repo: repo, analytics: analytics}
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
