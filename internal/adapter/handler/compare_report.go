package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/service"
)

// compareReportStore is satisfied by *db.CompareReportRepository.
type compareReportStore interface {
	Create(ctx context.Context, tenantID uuid.UUID, source string, ready bool, report any) (uuid.UUID, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*db.StoredCompareReport, error)
	List(ctx context.Context, tenantID uuid.UUID, limit int) ([]*db.StoredCompareReport, error)
}

// tenantNamer resolves the tenant display name for the printed document.
type tenantNamer interface {
	TenantName(ctx context.Context, tenantID uuid.UUID) string
}

// CompareReportHandler serves persisted migration Compare runs: the list, the
// raw report, and the printable document — each run a citable receipt that a
// migration was proven before cut-over.
type CompareReportHandler struct {
	store compareReportStore
	namer tenantNamer
}

func NewCompareReportHandler(store compareReportStore) *CompareReportHandler {
	return &CompareReportHandler{store: store}
}

// SetTenantNamer wires the display-name lookup. Nil-safe.
func (h *CompareReportHandler) SetTenantNamer(n tenantNamer) { h.namer = n }

// List returns the tenant's stored runs, newest first, envelope only (id,
// source, ready, generated_at) — the full report comes from GetByID.
// GET /v1/import/compare-reports
func (h *CompareReportHandler) List(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	// Caller-controllable page size (was a hardcoded 50 with no way to page).
	limit, _ := parseLimitOffset(c, 50, 200)
	rows, err := h.store.List(c.Request.Context(), tenantID, limit)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	type item struct {
		ID          uuid.UUID `json:"id"`
		Source      string    `json:"source"`
		Ready       bool      `json:"ready"`
		GeneratedAt string    `json:"generated_at"`
	}
	out := make([]item, 0, len(rows))
	for _, r := range rows {
		out = append(out, item{ID: r.ID, Source: r.Source, Ready: r.Ready, GeneratedAt: r.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// Get returns one stored run with its full report.
// GET /v1/import/compare-reports/:id
func (h *CompareReportHandler) Get(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid report id")
		return
	}
	rec, err := h.store.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if rec == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "report not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

// Document renders the printable receipt (print-to-PDF, like the invoice
// document flow).
// GET /v1/import/compare-reports/:id/document
func (h *CompareReportHandler) Document(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid report id")
		return
	}
	rec, err := h.store.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if rec == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "report not found")
		return
	}
	var report service.CompareReport
	if err := json.Unmarshal(rec.Report, &report); err != nil {
		respondInternalError(c, err)
		return
	}
	name := ""
	if h.namer != nil {
		name = h.namer.TenantName(c.Request.Context(), tenantID)
	}
	html, err := service.RenderCompareReportHTML(service.CompareReportDocData{
		TenantName:  name,
		Source:      rec.Source,
		Ready:       rec.Ready,
		GeneratedAt: rec.GeneratedAt,
		Report:      report,
	})
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\"compare-report-"+rec.Source+".html\"")
	c.String(http.StatusOK, html)
}
