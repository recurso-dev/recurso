package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// AuditHandler exposes the append-only audit trail (Lago-parity C2):
//
//	GET /v1/audit-logs   filters: entity_type, entity_id, actor, from, to,
//	                     limit, offset
type AuditHandler struct {
	repo port.AuditLogRepository
}

func NewAuditHandler(repo port.AuditLogRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) List(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	filter := domain.AuditLogFilter{
		EntityType: c.Query("entity_type"),
		EntityID:   c.Query("entity_id"),
		Actor:      c.Query("actor"),
	}
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "from must be RFC3339")
			return
		}
		filter.From = t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "to must be RFC3339")
			return
		}
		filter.To = t
	}
	filter.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "100"))
	filter.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	filter.Limit, filter.Offset = clampLimitOffset(filter.Limit, filter.Offset, 100, 250)

	logs, err := h.repo.List(ctx, tenantID, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to list audit logs")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}
