package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
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
	if status != "" &&
		status != string(domain.DisputeStatusOpen) &&
		status != string(domain.DisputeStatusResolved) &&
		status != string(domain.DisputeStatusRejected) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "status must be 'open', 'resolved' or 'rejected'")
		return
	}

	limit, offset := parseLimitOffset(c, 1000, 1000)
	disputes, err := h.service.List(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": disputes})
}

type resolveDisputeRequest struct {
	Note string `json:"note"`
	// Outcome is "accept" (default — closes in the customer's favor) or
	// "reject". Omitted/empty keeps the pre-outcome behavior (accept).
	Outcome string `json:"outcome"`
	// IssueCredit (accept only) issues an adjustment credit note against the
	// disputed invoice. CreditAmount is minor units; 0 = the invoice's amount due.
	IssueCredit  bool  `json:"issue_credit"`
	CreditAmount int64 `json:"credit_amount"`
}

// ResolveDispute handles POST /v1/disputes/:id/resolve (tenant-scoped): records
// the operator's decision (accept/reject) and, on accept, optionally issues a
// resolution credit note.
func (h *DisputeHandler) ResolveDispute(c *gin.Context) {
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid dispute id")
		return
	}

	// The body is optional, so an empty body is allowed (defaults to accept).
	var req resolveDisputeRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if req.Outcome != "" && req.Outcome != "accept" && req.Outcome != "reject" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "outcome must be 'accept' or 'reject'")
		return
	}
	// A negative credit would mint a credit note that debits the customer — never
	// valid. 0 is allowed (means "the invoice's amount due"); reject <0 at the edge.
	if req.CreditAmount < 0 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "credit_amount must be zero or positive")
		return
	}

	credit, err := h.service.ResolveWithOutcome(
		c.Request.Context(), tenantID, middleware.GetUserID(c), func() string {
			r, _ := middleware.GetUserRole(c)
			return r
		}(), id, service.DisputeResolution{
			Accept:       req.Outcome != "reject",
			Note:         req.Note,
			IssueCredit:  req.IssueCredit,
			CreditAmount: req.CreditAmount,
		})
	if err != nil {
		if err == domain.ErrDisputeNotFound {
			respondError(c, http.StatusNotFound, codeNotFound, "dispute not found")
			return
		}
		respondInternalError(c, err)
		return
	}

	status := "resolved"
	if req.Outcome == "reject" {
		status = "rejected"
	}
	resp := gin.H{"status": status}
	if credit != nil {
		resp["credit_note"] = credit
	}
	c.JSON(http.StatusOK, resp)
}
