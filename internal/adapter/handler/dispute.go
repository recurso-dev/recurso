package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

// DisputeHandler exposes the admin-facing (API-key) dispute endpoints.
// NOTE: there is intentionally no admin dashboard UI for disputes yet — the
// admin dashboard surface is a follow-up owned by another workstream.
type DisputeHandler struct {
	service *service.DisputeService
}

func NewDisputeHandler(s *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{service: s}
}

// ListDisputes handles GET /v1/disputes?status=open|resolved (tenant-scoped).
func (h *DisputeHandler) ListDisputes(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	status := c.Query("status")
	if status != "" && status != string(domain.DisputeStatusOpen) && status != string(domain.DisputeStatusResolved) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "status must be 'open' or 'resolved'")
		return
	}

	disputes, err := h.service.List(c.Request.Context(), tenantID, status)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": disputes})
}

type resolveDisputeRequest struct {
	Note string `json:"note"`
}

// ResolveDispute handles POST /v1/disputes/:id/resolve (tenant-scoped).
func (h *DisputeHandler) ResolveDispute(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid dispute id")
		return
	}

	// The note is optional, so an empty body is allowed.
	var req resolveDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if err := h.service.Resolve(c.Request.Context(), tenantID, id, req.Note); err != nil {
		if err == domain.ErrDisputeNotFound {
			respondError(c, http.StatusNotFound, codeNotFound, "dispute not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resolved"})
}
