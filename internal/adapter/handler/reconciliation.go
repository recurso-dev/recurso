package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/service"
)

// ReconciliationHandler exposes on-demand ledger reconciliation.
type ReconciliationHandler struct {
	service *service.ReconciliationService
}

func NewReconciliationHandler(service *service.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{service: service}
}

// RunReconciliation runs a ledger-vs-billing reconciliation for the caller's
// tenant and returns the computed report. Nothing is persisted.
func (h *ReconciliationHandler) RunReconciliation(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}

	report, err := h.service.Run(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to run reconciliation")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// RecordReconciliation runs a reconciliation AND records a summary of it to the
// run history (the audit trail), then returns the full report. The ephemeral
// GET stays side-effect-free; this is the explicit "run and record" action.
// POST /v1/finance/reconciliation/runs
func (h *ReconciliationHandler) RecordReconciliation(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	tid := tenantID.(uuid.UUID)

	report, err := h.service.Run(c.Request.Context(), tid)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to run reconciliation")
		return
	}
	// Best-effort: a persistence failure must not fail the reconciliation the
	// operator just ran — the report is still returned.
	_ = h.service.RecordRun(c.Request.Context(), tid, middleware.GetUserID(c), report)

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ListReconciliationRuns returns the tenant's recorded reconciliation runs,
// newest first — "when was it checked, by whom, and did it tie out?".
// GET /v1/finance/reconciliation/runs?limit=
func (h *ReconciliationHandler) ListReconciliationRuns(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	runs, err := h.service.ListRuns(c.Request.Context(), tenantID.(uuid.UUID), limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to list reconciliation runs")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": runs})
}

// GetReconciliationRun returns one recorded run with its stored discrepancy
// rows — the addressable, explainable run object. Cross-tenant ids return 404.
// GET /v1/finance/reconciliation/runs/:id
func (h *ReconciliationHandler) GetReconciliationRun(c *gin.Context) {
	tenantID, ok := c.Get("tenant_id")
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid reconciliation run id")
		return
	}
	run, err := h.service.GetRun(c.Request.Context(), tenantID.(uuid.UUID), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch reconciliation run")
		return
	}
	if run == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "reconciliation run not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": run})
}
