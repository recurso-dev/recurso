package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// collectionsQueueLister is the read-only slice of the invoice repository the
// collections worklist needs. *db.InvoiceRepository satisfies it.
type collectionsQueueLister interface {
	ListCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) ([]domain.CollectionsQueueItem, error)
	CountCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) (int, error)
}

// CollectionsHandler serves the operator-facing collections views
// (Collections Intelligence). Read-only; the automated recovery engine is
// untouched.
type CollectionsHandler struct {
	repo collectionsQueueLister
}

func NewCollectionsHandler(repo collectionsQueueLister) *CollectionsHandler {
	return &CollectionsHandler{repo: repo}
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
