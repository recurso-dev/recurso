package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/middleware"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TaxNexusHandler serves the US sales-tax nexus configuration
// (GET/PUT /v1/settings/tax/nexus).
type TaxNexusHandler struct {
	repo *db.TaxNexusRepository
}

func NewTaxNexusHandler(repo *db.TaxNexusRepository) *TaxNexusHandler {
	return &TaxNexusHandler{repo: repo}
}

// requireManager gates writes to owner/admin; API-key (machine) callers pass.
func (h *TaxNexusHandler) requireManager(c *gin.Context) bool {
	role, hasUser := middleware.GetUserRole(c)
	if !hasUser {
		return true
	}
	if domain.Role(role).CanManageTeam() {
		return true
	}
	respondError(c, http.StatusForbidden, codeForbidden, "requires owner or admin role")
	return false
}

type nexusStateItem struct {
	StateCode string `json:"state_code" binding:"required"`
	NexusType string `json:"nexus_type"`
}

type setNexusRequest struct {
	States []nexusStateItem `json:"states"`
}

// GetNexus lists the tenant's declared US nexus states.
func (h *TaxNexusHandler) GetNexus(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	list, err := h.repo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if list == nil {
		list = []domain.TaxNexus{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// SetNexus replaces the tenant's entire nexus set (owner/admin only).
func (h *TaxNexusHandler) SetNexus(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if !h.requireManager(c) {
		return
	}

	var req setNexusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	states := make([]domain.TaxNexus, 0, len(req.States))
	for _, s := range req.States {
		code := strings.ToUpper(strings.TrimSpace(s.StateCode))
		if len(code) != 2 {
			respondError(c, http.StatusBadRequest, codeValidationFailed,
				"state_code must be a two-letter US state code: "+s.StateCode)
			return
		}
		nt := domain.NexusType(strings.ToLower(strings.TrimSpace(s.NexusType)))
		switch nt {
		case domain.NexusPhysical, domain.NexusVoluntary, domain.NexusEconomic:
		case "":
			nt = domain.NexusPhysical
		default:
			respondError(c, http.StatusBadRequest, codeValidationFailed,
				"nexus_type must be physical, voluntary, or economic")
			return
		}
		states = append(states, domain.TaxNexus{StateCode: code, NexusType: nt})
	}

	if err := h.repo.SetStates(c.Request.Context(), tenantID, states); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	list, err := h.repo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if list == nil {
		list = []domain.TaxNexus{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "message": "nexus states updated"})
}
