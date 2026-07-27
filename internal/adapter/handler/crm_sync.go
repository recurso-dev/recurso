package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/worker"
)

// crmTenantSyncer is the worker slice this handler needs; *worker.CRMSyncWorker.
type crmTenantSyncer interface {
	RunTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// CRMSyncHandler exposes the manual "sync now" for a tenant's CRM connection —
// how a fresh HubSpot token gets tested without waiting for the daily sweep.
type CRMSyncHandler struct {
	syncer crmTenantSyncer // nil when the CRM worker isn't running
}

func NewCRMSyncHandler(syncer crmTenantSyncer) *CRMSyncHandler {
	return &CRMSyncHandler{syncer: syncer}
}

// SyncNow handles POST /v1/crm/sync — runs this tenant's CRM sweep
// synchronously and reports how many contacts were upserted.
func (h *CRMSyncHandler) SyncNow(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	if h.syncer == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "CRM sync is not enabled on this server")
		return
	}
	synced, err := h.syncer.RunTenant(c.Request.Context(), tenantID)
	switch {
	case errors.Is(err, worker.ErrCRMNotConfigured):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	case err != nil:
		// The provider's own rejection (401 bad token, 403 missing scopes) is
		// exactly what the operator needs to see when testing a connection.
		respondError(c, http.StatusBadGateway, codeInternalError, "CRM sync failed: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"contacts_synced": synced}})
}
