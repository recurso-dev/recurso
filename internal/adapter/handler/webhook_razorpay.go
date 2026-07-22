package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

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
		// Order is present on order.paid events. Mandate debits create the
		// Razorpay order with notes.invoice_id, but the auto-debited payment
		// entity does not inherit those notes — the order entity is the only
		// place the invoice id appears for mandate-collected payments.
		Order struct {
			Entity struct {
				ID    string `json:"id"`
				Notes struct {
					InvoiceID string `json:"invoice_id"`
				} `json:"notes"`
			} `json:"entity"`
		} `json:"order"`
	} `json:"payload"`
}

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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read body")
		return
	}

	// 1. Verify Signature. Per-connection route (:connID) verifies with that
	// tenant's own webhook secret; the legacy route uses the env secret. Fail
	// CLOSED either way: an unconfigured secret must reject the webhook, not
	// process it. Otherwise a forged payment.captured with a known invoice_id
	// would mark an invoice paid on a misconfigured deploy (ENG-145).
	signature := c.GetHeader("X-Razorpay-Signature")
	envSecret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")
	if envSecret == "" && c.Param("connID") == "" {
		h.logger.Error("RAZORPAY_WEBHOOK_SECRET not set — rejecting webhook (fail closed)", "ip", c.ClientIP())
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "webhook verification not configured")
		return
	}
	webhookSecret, ok := h.webhookSecretFor(c, domain.GatewayRazorpay, envSecret)
	if !ok {
		return
	}
	if !verifyRazorpaySignature(body, signature, webhookSecret) {
		h.logger.Warn("webhook signature verification failed",
			"signature", signature,
			"ip", c.ClientIP(),
		)
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid signature")
		return
	}

	var event RazorpayWebhookPayload
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("invalid webhook JSON", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid JSON")
		return
	}

	h.logger.Info("webhook received", "event", event.Event)

	// Idempotency: skip an event we've already fully processed. Razorpay sends a
	// unique X-Razorpay-Event-Id header per event; a redelivery reuses it, and
	// without this a duplicate re-runs non-idempotent side effects (the
	// payment-failed email, the dunning bandit outcome). Recorded only after a
	// 2xx below, so a failed delivery is still retried (ENG-162).
	eventID := c.GetHeader("X-Razorpay-Event-Id")
	if h.alreadyProcessed(c, "razorpay", eventID) {
		return
	}

	// Every case writes its own response; the tail records the event as
	// processed only when that response was 2xx.
	switch event.Event {
	case "token.confirmed": // UPI mandate authorization
		h.handleTokenConfirmed(c, body)
	case "virtual_account.credited": // offline payment reconciliation
		h.handleVirtualAccountCredited(c, body)
	case "payment.failed":
		h.handleRazorpayPaymentFailed(c, event)
	case "refund.processed", "refund.failed": // advance the credit note tracking the refund
		h.handleRazorpayRefundEvent(c, body, event.Event == "refund.processed")
	case "payment.captured", "order.paid":
		h.handleRazorpayPaymentCaptured(c, event)
	default:
		h.logger.Info("razorpay webhook event ignored", "event", event.Event)
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
	}

	// Record only on a 2xx so a failed delivery (5xx) is retried by Razorpay and
	// reprocessed (ENG-162).
	if status := c.Writer.Status(); status >= 200 && status < 300 {
		h.markProcessed(c.Request.Context(), "razorpay", eventID, event.Event)
	}
}

// handleRazorpayPaymentCaptured settles the invoice referenced by a
// payment.captured / order.paid event and writes the HTTP response.
func (h *WebhookHandler) handleRazorpayPaymentCaptured(c *gin.Context, event RazorpayWebhookPayload) {
	invoiceIDStr := event.Payload.Payment.Entity.Notes.InvoiceID
	if invoiceIDStr == "" {
		// Mandate-debit payments carry no notes of their own; on order.paid
		// the order entity holds the notes.invoice_id set when the debit
		// order was created (see MandateService.ExecuteDebit).
		invoiceIDStr = event.Payload.Order.Entity.Notes.InvoiceID
	}
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

	ctx := c.Request.Context()

	// MarkInvoicePaid reads the invoice through the tenant-scoped repository,
	// and webhook requests carry no tenant — load the invoice first and inject
	// its own tenant id. Already-paid invoices (e.g. mandate debits) no-op
	// inside MarkInvoicePaid, so redelivery is safe.
	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		h.logger.Error("failed to load invoice for payment webhook", "invoice_id", invoiceID, "error", err)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to load invoice")
		return
	}
	if inv == nil {
		h.logger.Warn("webhook ignored — invoice not found", "invoice_id", invoiceID)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "unknown invoice_id"})
		return
	}

	ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
	transitioned, err := h.subService.MarkInvoicePaid(ctxWithTenant, invoiceID)
	if err != nil {
		h.logger.Error("failed to mark invoice paid", "invoice_id", invoiceID, "error", err)
		respondInternalError(c, err)
		return
	}

	h.logger.Info("invoice marked paid via webhook", "invoice_id", invoiceID)

	// Persist the gateway payment id (pay_*) — refunds are issued against it. For
	// mandate-collected invoices this is the only place the actual payment id
	// becomes known: the debit response returns an order id.
	//
	// Only store a real payment id: order.paid can carry an order_* (or empty)
	// value here, and writing that would poison gateway_payment_id so refunds
	// fail (RazorpayGateway.Refund rejects non-pay_*). ExecuteDebit guards the
	// same write for this reason; the webhook path did not (ENG-188).
	if paymentID := event.Payload.Payment.Entity.ID; isGatewayPaymentID(paymentID) && h.invoiceRepo != nil {
		if err := h.invoiceRepo.SetGatewayPaymentID(ctx, inv.TenantID, invoiceID, paymentID); err != nil {
			h.logger.Error("failed to record gateway payment id",
				"invoice_id", invoiceID, "payment_id", paymentID, "error", err)
		}
	}

	// Record the smart-dunning success only when THIS delivery performed the paid
	// transition, so a redelivered webhook (or a second settler) can't
	// double-count the dunning bandit's reward (ENG-162).
	if transitioned {
		h.recordDunningSuccess(ctx, invoiceID)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleRazorpayRefundEvent consumes refund.processed / refund.failed and
// advances the credit note owning the refund (pending → processed /
// refund_failed). Refund ids that match no credit note are logged and
// acknowledged with 200 — Razorpay retries (and eventually disables) webhooks
// on non-2xx responses, and a refund we did not issue can never be resolved.
func (h *WebhookHandler) handleRazorpayRefundEvent(c *gin.Context, body []byte, succeeded bool) {
	if h.creditNoteService == nil {
		h.logger.Info("credit note service not configured, ignoring refund event")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	var payload struct {
		Payload struct {
			Refund struct {
				Entity struct {
					ID        string `json:"id"`
					PaymentID string `json:"payment_id"`
					Status    string `json:"status"`
				} `json:"entity"`
			} `json:"refund"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse razorpay refund payload", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid payload")
		return
	}

	refundID := payload.Payload.Refund.Entity.ID
	if refundID == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no refund_id"})
		return
	}

	reason := ""
	if !succeeded {
		// Razorpay's refund entity carries no failure description; record what
		// the event does tell us.
		reason = fmt.Sprintf("razorpay reported the refund as failed (payment %s, refund status %q)",
			payload.Payload.Refund.Entity.PaymentID, payload.Payload.Refund.Entity.Status)
	}

	if err := h.creditNoteService.ProcessGatewayRefundEvent(c.Request.Context(), refundID, succeeded, reason); err != nil {
		if errors.Is(err, service.ErrRefundNotFound) {
			h.logger.Info("razorpay refund event ignored — no matching credit note", "refund_id", refundID)
			c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "unknown refund_id"})
			return
		}
		h.logger.Error("failed to process razorpay refund event", "refund_id", refundID, "error", err)
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handleTokenConfirmed(c *gin.Context, body []byte) {
	if h.mandateService == nil {
		h.logger.Info("mandate service not configured, ignoring token.confirmed")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	var payload struct {
		Payload struct {
			Token struct {
				Entity struct {
					ID         string `json:"id"`
					CustomerID string `json:"customer_id"`
				} `json:"entity"`
			} `json:"token"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse token.confirmed payload", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid payload")
		return
	}

	tokenID := payload.Payload.Token.Entity.ID
	if tokenID == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no token_id"})
		return
	}

	customerID := payload.Payload.Token.Entity.CustomerID
	if err := h.mandateService.HandleAuthorization(c.Request.Context(), tokenID, customerID); err != nil {
		h.logger.Error("failed to handle mandate authorization", "token_id", tokenID, "error", err)
		respondInternalError(c, err)
		return
	}

	h.logger.Info("mandate authorized via webhook", "token_id", tokenID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handleVirtualAccountCredited(c *gin.Context, body []byte) {
	if h.offlinePaymentSvc == nil {
		h.logger.Info("offline payment service not configured, ignoring virtual_account.credited")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	var payload struct {
		Payload struct {
			VirtualAccount struct {
				Entity struct {
					ID             string `json:"id"`
					AmountPaid     int64  `json:"amount_paid"`
					AmountExpected int64  `json:"amount_expected"`
				} `json:"entity"`
			} `json:"virtual_account"`
			Payment struct {
				Entity struct {
					ID     string `json:"id"`
					Amount int64  `json:"amount"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse virtual_account.credited payload", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid payload")
		return
	}

	vaID := payload.Payload.VirtualAccount.Entity.ID
	amount := payload.Payload.Payment.Entity.Amount
	paymentID := payload.Payload.Payment.Entity.ID

	if vaID == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no va_id"})
		return
	}

	if err := h.offlinePaymentSvc.ReconcileVirtualAccount(c.Request.Context(), vaID, amount, paymentID); err != nil {
		h.logger.Error("failed to reconcile virtual account", "va_id", vaID, "error", err)
		respondInternalError(c, err)
		return
	}

	h.logger.Info("virtual account payment reconciled via webhook", "va_id", vaID, "amount", amount)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handleRazorpayPaymentFailed(c *gin.Context, event RazorpayWebhookPayload) {
	invoiceIDStr := event.Payload.Payment.Entity.Notes.InvoiceID
	if invoiceIDStr == "" {
		h.logger.Info("razorpay payment.failed ignored — no invoice_id in notes",
			"payment_id", event.Payload.Payment.Entity.ID,
		)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "no invoice_id"})
		return
	}

	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.logger.Warn("invalid invoice_id in razorpay webhook", "invoice_id", invoiceIDStr)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "reason": "invalid invoice_id"})
		return
	}

	ctx := c.Request.Context()

	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil || inv == nil {
		h.logger.Error("failed to fetch invoice for razorpay payment failure",
			"invoice_id", invoiceID,
			"error", err,
		)
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}

	inv.Status = domain.InvoiceStatusPastDue
	if err := h.invoiceRepo.Update(ctx, inv); err != nil {
		h.logger.Error("failed to update invoice to past_due via razorpay",
			"invoice_id", invoiceID,
			"error", err,
		)
	}

	h.recordDunningFailure(ctx, invoiceID)

	// Trigger dunning campaign
	if h.dunningCampaignService != nil {
		if err := h.dunningCampaignService.TriggerCampaign(ctx, invoiceID, "payment_failed"); err != nil {
			h.logger.Error("failed to trigger dunning campaign via razorpay", "error", err, "invoice_id", invoiceID)
		}
	}

	h.logger.Info("razorpay payment failure processed", "invoice_id", invoiceID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
