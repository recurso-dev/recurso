package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// UsageAlertHandler exposes usage threshold alerts (Lago-parity B3):
//
//	POST   /v1/usage-alerts                create an alert
//	GET    /v1/usage-alerts                list (optional ?subscription_id=)
//	DELETE /v1/usage-alerts/:id            delete an alert
type UsageAlertHandler struct {
	svc *service.UsageAlertService
}

func NewUsageAlertHandler(svc *service.UsageAlertService) *UsageAlertHandler {
	return &UsageAlertHandler{svc: svc}
}

func alertTenantCtx(c *gin.Context) (uuid.UUID, context.Context, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, nil, false
	}
	return tenantID, context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

func respondAlertError(c *gin.Context, err error) {
	var valErr service.MeteringValidationError
	switch {
	case errors.Is(err, service.ErrAlertNotFound), errors.Is(err, service.ErrUsageSubscriptionNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, service.ErrAlertExists):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.As(err, &valErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}

func (h *UsageAlertHandler) Create(c *gin.Context) {
	tenantID, ctx, ok := alertTenantCtx(c)
	if !ok {
		return
	}
	var in service.UsageAlertInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	a, err := h.svc.CreateAlert(ctx, tenantID, in)
	if err != nil {
		respondAlertError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": a})
}

func (h *UsageAlertHandler) List(c *gin.Context) {
	tenantID, ctx, ok := alertTenantCtx(c)
	if !ok {
		return
	}
	var subID *uuid.UUID
	if raw := c.Query("subscription_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription_id")
			return
		}
		subID = &id
	}
	alerts, err := h.svc.ListAlerts(ctx, tenantID, subID)
	if err != nil {
		respondAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

func (h *UsageAlertHandler) Delete(c *gin.Context) {
	tenantID, ctx, ok := alertTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid alert id")
		return
	}
	if err := h.svc.DeleteAlert(ctx, tenantID, id); err != nil {
		respondAlertError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
