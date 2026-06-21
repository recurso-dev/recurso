package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

type WebhookHandler struct {
	subService          *service.SubscriptionService
	gateway             port.PaymentGateway
	retryService        *service.SmartRetryService
	invoiceRepo         port.InvoiceRepository
	subRepo             port.SubscriptionRepository
	stripeWebhookSecret string
	logger              *slog.Logger
}

func NewWebhookHandler(
	subService *service.SubscriptionService,
	gateway port.PaymentGateway,
	retryService *service.SmartRetryService,
	invoiceRepo port.InvoiceRepository,
	subRepo port.SubscriptionRepository,
	stripeWebhookSecret string,
) *WebhookHandler {
	return &WebhookHandler{
		subService:          subService,
		gateway:             gateway,
		retryService:        retryService,
		invoiceRepo:         invoiceRepo,
		subRepo:             subRepo,
		stripeWebhookSecret: stripeWebhookSecret,
		logger:              slog.Default().With("component", "webhook_handler"),
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

// HandleStripe handles incoming Stripe webhook events.
func (h *WebhookHandler) HandleStripe(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read stripe webhook body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 1. Verify Signature
	var event stripe.Event
	if h.stripeWebhookSecret != "" {
		event, err = webhook.ConstructEvent(body, c.GetHeader("Stripe-Signature"), h.stripeWebhookSecret)
		if err != nil {
			h.logger.Warn("stripe webhook signature verification failed",
				"error", err,
				"ip", c.ClientIP(),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	} else {
		h.logger.Warn("STRIPE_WEBHOOK_SECRET not set — skipping signature verification")
		if err := json.Unmarshal(body, &event); err != nil {
			h.logger.Error("invalid stripe webhook JSON", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
	}

	h.logger.Info("stripe webhook received", "event_type", event.Type)
	ctx := c.Request.Context()

	var handlerErr error
	switch event.Type {
	case "payment_intent.succeeded":
		handlerErr = h.handlePaymentIntentSucceeded(ctx, event)
	case "invoice.payment_failed":
		handlerErr = h.handleInvoicePaymentFailed(ctx, event)
	case "customer.subscription.deleted":
		handlerErr = h.handleSubscriptionDeleted(ctx, event)
	default:
		h.logger.Info("stripe webhook event ignored", "event_type", event.Type)
	}

	if handlerErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": handlerErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.logger.Error("failed to unmarshal payment intent", "error", err)
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}

	invoiceIDStr := pi.Metadata["invoice_id"]
	if invoiceIDStr == "" {
		h.logger.Info("stripe payment_intent.succeeded ignored — no invoice_id in metadata",
			"payment_intent_id", pi.ID,
		)
		return nil
	}

	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.logger.Warn("invalid invoice_id in stripe metadata", "invoice_id", invoiceIDStr)
		return nil
	}

	if err := h.subService.MarkInvoicePaid(ctx, invoiceID); err != nil {
		h.logger.Error("failed to mark invoice paid via stripe webhook",
			"invoice_id", invoiceID,
			"error", err,
		)
		return fmt.Errorf("failed to mark invoice paid: %w", err)
	}

	h.logger.Info("invoice marked paid via stripe webhook", "invoice_id", invoiceID)
	h.recordDunningSuccess(ctx, invoiceID)
	return nil
}

func (h *WebhookHandler) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	var stripeInvoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &stripeInvoice); err != nil {
		h.logger.Error("failed to unmarshal stripe invoice", "error", err)
		return fmt.Errorf("failed to unmarshal stripe invoice: %w", err)
	}

	invoiceIDStr := stripeInvoice.Metadata["invoice_id"]
	if invoiceIDStr == "" {
		h.logger.Info("stripe invoice.payment_failed ignored — no invoice_id in metadata",
			"stripe_invoice_id", stripeInvoice.ID,
		)
		return nil
	}

	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		h.logger.Warn("invalid invoice_id in stripe invoice metadata", "invoice_id", invoiceIDStr)
		return nil
	}

	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil || inv == nil {
		h.logger.Error("failed to fetch invoice for payment failure",
			"invoice_id", invoiceID,
			"error", err,
		)
		return fmt.Errorf("failed to fetch invoice %s: %w", invoiceID, err)
	}

	inv.Status = domain.InvoiceStatusPastDue
	inv.LastPaymentError = stripeInvoice.Metadata["error_message"]
	if inv.LastPaymentError == "" && stripeInvoice.LastFinalizationError != nil {
		inv.LastPaymentError = stripeInvoice.LastFinalizationError.Msg
	}

	if err := h.invoiceRepo.Update(ctx, inv); err != nil {
		h.logger.Error("failed to update invoice to past_due",
			"invoice_id", invoiceID,
			"error", err,
		)
		return fmt.Errorf("failed to update invoice to past_due: %w", err)
	}

	h.logger.Info("invoice marked past_due via stripe webhook", "invoice_id", invoiceID)
	h.recordDunningFailure(ctx, invoiceID)
	return nil
}

func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		h.logger.Error("failed to unmarshal stripe subscription", "error", err)
		return fmt.Errorf("failed to unmarshal stripe subscription: %w", err)
	}

	if h.subRepo == nil {
		h.logger.Error("subscription repository not available for stripe webhook")
		return fmt.Errorf("subscription repository not configured")
	}

	sub, err := h.subRepo.GetByStripeSubscriptionID(ctx, stripeSub.ID)
	if err != nil {
		h.logger.Error("failed to find subscription by stripe ID",
			"stripe_subscription_id", stripeSub.ID,
			"error", err,
		)
		return fmt.Errorf("failed to find subscription by stripe ID %s: %w", stripeSub.ID, err)
	}

	// Use Cancel with tenant context
	ctxWithTenant := context.WithValue(ctx, "tenant_id", sub.TenantID)
	_, err = h.subService.Cancel(ctxWithTenant, sub.TenantID, sub.ID, true, "stripe_webhook", "subscription deleted in Stripe")
	if err != nil {
		h.logger.Error("failed to cancel subscription via stripe webhook",
			"subscription_id", sub.ID,
			"stripe_subscription_id", stripeSub.ID,
			"error", err,
		)
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	h.logger.Info("subscription canceled via stripe webhook",
		"subscription_id", sub.ID,
		"stripe_subscription_id", stripeSub.ID,
	)
	return nil
}

// recordDunningFailure records a reward=0.0 outcome if the invoice has an active dunning action
func (h *WebhookHandler) recordDunningFailure(ctx context.Context, invoiceID uuid.UUID) {
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
		Outcome:       "failure",
		Reward:        0.0,
		CreatedAt:     time.Now(),
	})
	if err != nil {
		h.logger.Error("failed to record dunning failure outcome",
			"invoice_id", invoiceID,
			"error", err,
		)
	} else {
		h.logger.Info("recorded dunning failure outcome via stripe webhook",
			"invoice_id", invoiceID,
			"action_id", inv.DunningActionID,
			"context_key", inv.DunningContextKey,
		)
	}
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
