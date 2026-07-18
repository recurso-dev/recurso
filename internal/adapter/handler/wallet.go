package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// WalletHandler exposes prepaid wallets (Lago-parity B1):
//
//	POST /v1/wallets                       create a wallet
//	GET  /v1/wallets/:id                   fetch one wallet
//	GET  /v1/customers/:id/wallets         a customer's wallets
//	POST /v1/wallets/:id/top-up            add balance (manual/promotional)
//	GET  /v1/wallets/:id/transactions      movement history
//	PUT  /v1/wallets/:id/auto-recharge     set/clear the recharge rule
type WalletHandler struct {
	svc *service.WalletService
}

func NewWalletHandler(svc *service.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

func walletTenantCtx(c *gin.Context) (uuid.UUID, context.Context, bool) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return uuid.Nil, nil, false
	}
	return tenantID, context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

func respondWalletError(c *gin.Context, err error) {
	var valErr service.WalletValidationError
	switch {
	case errors.Is(err, service.ErrWalletNotFound), errors.Is(err, service.ErrWalletCustomerGone):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, service.ErrWalletExists):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
	case errors.As(err, &valErr):
		respondError(c, http.StatusBadRequest, codeValidationFailed, valErr.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
}

func (h *WalletHandler) Create(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	var in service.CreateWalletInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	w, err := h.svc.CreateWallet(ctx, tenantID, in)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": w})
}

func (h *WalletHandler) Get(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid wallet id")
		return
	}
	w, err := h.svc.GetWallet(ctx, tenantID, id)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}

func (h *WalletHandler) ListForCustomer(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}
	wallets, err := h.svc.ListCustomerWallets(ctx, tenantID, customerID)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": wallets})
}

func (h *WalletHandler) TopUp(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid wallet id")
		return
	}
	var in service.TopUpInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	wtx, err := h.svc.TopUp(ctx, tenantID, id, in)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": wtx})
}

func (h *WalletHandler) ListTransactions(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid wallet id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	txs, err := h.svc.ListTransactions(ctx, tenantID, id, limit)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": txs})
}

type autoRechargeRequest struct {
	Threshold *int64 `json:"auto_recharge_threshold"`
	Amount    *int64 `json:"auto_recharge_amount"`
}

func (h *WalletHandler) UpdateAutoRecharge(c *gin.Context) {
	tenantID, ctx, ok := walletTenantCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid wallet id")
		return
	}
	var req autoRechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	if err := h.svc.UpdateAutoRecharge(ctx, tenantID, id, req.Threshold, req.Amount); err != nil {
		respondWalletError(c, err)
		return
	}
	w, err := h.svc.GetWallet(ctx, tenantID, id)
	if err != nil {
		respondWalletError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}
