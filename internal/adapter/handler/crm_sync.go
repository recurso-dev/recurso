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
	RunTenant(ctx context.Context, tenantID uuid.UUID, maxContacts int) (int, int, error)
}

// manualSyncBatch caps how many contacts one "Sync now" click pushes. The
// click exists to VERIFY a connection; an unbounded sweep can outlive the
// CDN's ~100s proxy timeout on big tenants and the browser sees a bare
// failure while the server keeps (partially) syncing. The daily sweep is
// the unbounded backfill.
const manualSyncBatch = 25

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
	synced, remaining, err := h.syncer.RunTenant(c.Request.Context(), tenantID, manualSyncBatch)
	switch {
	case errors.Is(err, worker.ErrCRMNotConfigured):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	case err != nil:
		// The provider's own rejection (401 bad token, 403 missing scopes) is
		// exactly what the operator needs to see when testing a connection.
		// 424, NOT 502: Cloudflare fronts the API and REPLACES origin 502/504
		// responses with its own HTML error page — the JSON detail would never
		// reach the browser (live-diagnosed: the toast fell back to a generic
		// message while the real error died at the edge).
		respondError(c, http.StatusFailedDependency, codeInternalError, "CRM sync failed: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"contacts_synced": synced, "contacts_remaining": remaining}})
}
