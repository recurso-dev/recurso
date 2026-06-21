package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

type WebhookHandler struct {
	subService   *service.SubscriptionService
	gateway      port.PaymentGateway
	retryService *service.SmartRetryService
	invoiceRepo  port.InvoiceRepository
	logger       *slog.Logger
}

func NewWebhookHandler(
	subService *service.SubscriptionService,
	gateway port.PaymentGateway,
	retryService *service.SmartRetryService,
	invoiceRepo port.InvoiceRepository,
) *WebhookHandler {
	return &WebhookHandler{
		subService:   subService,
		gateway:      gateway,
		retryService: retryService,
		invoiceRepo:  invoiceRepo,
		logger:       slog.Default().With("component", "webhook_handler"),
	}
}

// RazorpayWebhookPayload is a simplified structure
type RazorpayWebhookPayload struct {
	Event   string `json:"event"`
	Payload struct {
		Payment struct {
			Entity struct {
				ID      string `json:"id"`
				OrderID string `json:"order_id"`
				Notes   struct {
					InvoiceID string `json:"invoice_id"`
				} `json:"notes"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

// verifyRazorpaySignature verifies the HMAC SHA256 signature from Razorpay webhooks
func verifyRazorpaySignature(body []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

func (h *WebhookHandler) HandleRazorpay(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 1. Verify Signature
	signature := c.GetHeader("X-Razorpay-Signature")
	webhookSecret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")

	if webhookSecret != "" {
		if !verifyRazorpaySignature(body, signature, webhookSecret) {
			h.logger.Warn("webhook signature verification failed",
				"signature", signature,
				"ip", c.ClientIP(),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else {
		h.logger.Warn("RAZORPAY_WEBHOOK_SECRET not set — skipping signature verification")
	}

	var event RazorpayWebhookPayload
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("invalid webhook JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	h.logger.Info("webhook received", "event", event.Event)

	if event.Event == "payment.captured" || event.Event == "order.paid" {
		invoiceIDStr := event.Payload.Payment.Entity.Notes.InvoiceID
		if invoiceIDStr == "" {
			h.logger.Info("webhook ignored — no invoice_id in notes",
				"event", event.Event,
				"payment_id", event.Payload.Payment.Entity.ID,
			)
			c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no invoice_id"})
			return
		}

		invoiceID, err := uuid.Parse(invoiceIDStr)
		if err != nil {
			h.logger.Warn("invalid invoice_id in webhook", "invoice_id", invoiceIDStr)
			c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "invalid invoice_id"})
			return
		}

		// 2. Mark Invoice as Paid
		if err := h.subService.MarkInvoicePaid(c.Request.Context(), invoiceID); err != nil {
			h.logger.Error("failed to mark invoice paid",
				"invoice_id", invoiceID,
				"error", err,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		h.logger.Info("invoice marked paid via webhook", "invoice_id", invoiceID)

		// 3. Record success outcome for RL if this invoice was managed by smart dunning
		h.recordDunningSuccess(c.Request.Context(), invoiceID)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// recordDunningSuccess records a reward=1.0 outcome if the invoice has an active dunning action
func (h *WebhookHandler) recordDunningSuccess(ctx context.Context, invoiceID uuid.UUID) {
	if h.retryService == nil || h.invoiceRepo == nil {
		return
	}

	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil || inv == nil {
		return
	}

	if inv.DunningActionID == "" || inv.DunningContextKey == "" {
		return
	}

	err = h.retryService.RecordOutcome(ctx, domain.DunningHistory{
		ID:            uuid.New(),
		TenantID:      inv.TenantID,
		InvoiceID:     inv.ID,
		ContextKey:    inv.DunningContextKey,
		ActionID:      inv.DunningActionID,
		RetryInterval: getDunningActionSeconds(inv.DunningActionID),
		Outcome:       "success",
		Reward:        1.0,
		CreatedAt:     time.Now(),
	})
	if err != nil {
		h.logger.Error("failed to record dunning success outcome",
			"invoice_id", invoiceID,
			"error", err,
		)
	} else {
		h.logger.Info("recorded dunning success outcome via webhook",
			"invoice_id", invoiceID,
			"action_id", inv.DunningActionID,
			"context_key", inv.DunningContextKey,
		)
	}
}

func getDunningActionSeconds(actionID string) int64 {
	for _, a := range domain.DefaultDunningActions {
		if a.ID == actionID {
			return int64(a.Interval.Seconds())
		}
	}
	return 86400
}
