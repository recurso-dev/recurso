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
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

type WebhookHandler struct {
	subService             *service.SubscriptionService
	gateway                port.PaymentGateway
	retryService           *service.SmartRetryService
	invoiceRepo            port.InvoiceRepository
	subRepo                port.SubscriptionRepository
	customerRepo           port.CustomerRepository
	notificationService    *service.NotificationService
	mandateService         *service.MandateService
	offlinePaymentSvc      *service.OfflinePaymentService
	dunningCampaignService *service.DunningCampaignService
	creditNoteService      *service.CreditNoteService
	inboundDedup           InboundWebhookDedup // nil-safe; when unset, dedup is skipped
	stripeWebhookSecret    string
	logger                 *slog.Logger
}

// InboundWebhookDedup records processed gateway webhook events so redeliveries
// can be acknowledged without re-running non-idempotent side effects.
// Satisfied by *db.InboundWebhookRepository.
type InboundWebhookDedup interface {
	WasProcessed(ctx context.Context, gateway, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, gateway, eventID, eventType string) error
}

// SetInboundWebhookDedup wires inbound webhook idempotency (ENG-162). Nil-safe:
// left unset, webhook processing is unchanged.
func (h *WebhookHandler) SetInboundWebhookDedup(d InboundWebhookDedup) { h.inboundDedup = d }

// alreadyProcessed acknowledges and returns true when this (gateway, eventID)
// was already fully processed — the caller should stop. Fails open (returns
// false) on a nil store, empty id, or a lookup error, so dedup never blocks a
// legitimate event.
func (h *WebhookHandler) alreadyProcessed(c *gin.Context, gateway, eventID string) bool {
	if h.inboundDedup == nil || eventID == "" {
		return false
	}
	processed, err := h.inboundDedup.WasProcessed(c.Request.Context(), gateway, eventID)
	if err != nil {
		h.logger.Error("webhook dedup check failed; processing anyway", "gateway", gateway, "event_id", eventID, "error", err)
		return false
	}
	if processed {
		h.logger.Info("duplicate webhook ignored", "gateway", gateway, "event_id", eventID)
		c.JSON(http.StatusOK, gin.H{"status": "duplicate ignored"})
		return true
	}
	return false
}

// markProcessed records a fully-processed event so redeliveries are skipped.
// Best-effort: a failure just means a possible reprocess later.
func (h *WebhookHandler) markProcessed(ctx context.Context, gateway, eventID, eventType string) {
	if h.inboundDedup == nil || eventID == "" {
		return
	}
	if err := h.inboundDedup.MarkProcessed(ctx, gateway, eventID, eventType); err != nil {
		h.logger.Error("failed to record processed webhook event", "gateway", gateway, "event_id", eventID, "error", err)
	}
}

func NewWebhookHandler(
	subService *service.SubscriptionService,
	gateway port.PaymentGateway,
	retryService *service.SmartRetryService,
	invoiceRepo port.InvoiceRepository,
	subRepo port.SubscriptionRepository,
	customerRepo port.CustomerRepository,
	notificationService *service.NotificationService,
	stripeWebhookSecret string,
) *WebhookHandler {
	return &WebhookHandler{
		subService:          subService,
		gateway:             gateway,
		retryService:        retryService,
		invoiceRepo:         invoiceRepo,
		subRepo:             subRepo,
		customerRepo:        customerRepo,
		notificationService: notificationService,
		stripeWebhookSecret: stripeWebhookSecret,
		logger:              slog.Default().With("component", "webhook_handler"),
	}
}

func (h *WebhookHandler) SetMandateService(svc *service.MandateService) {
	h.mandateService = svc
}

func (h *WebhookHandler) SetOfflinePaymentService(svc *service.OfflinePaymentService) {
	h.offlinePaymentSvc = svc
}

func (h *WebhookHandler) SetDunningCampaignService(svc *service.DunningCampaignService) {
	h.dunningCampaignService = svc
}

// SetCreditNoteService wires refund webhook consumption (Stripe
// charge.refunded / refund.failed, Razorpay refund.processed / refund.failed)
// so pending refund credit notes are advanced to processed / refund_failed.
func (h *WebhookHandler) SetCreditNoteService(svc *service.CreditNoteService) {
	h.creditNoteService = svc
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read body")
		return
	}

	// 1. Verify Signature
	signature := c.GetHeader("X-Razorpay-Signature")
	webhookSecret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")

	// Fail CLOSED: an unconfigured secret must reject the webhook, not process
	// it. Otherwise a forged payment.captured with a known invoice_id would mark
	// an invoice paid on a misconfigured deploy (ENG-145). Stripe already always
	// verifies.
	if webhookSecret == "" {
		h.logger.Error("RAZORPAY_WEBHOOK_SECRET not set — rejecting webhook (fail closed)", "ip", c.ClientIP())
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "webhook verification not configured")
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

	// Handle token.confirmed for UPI mandate authorization
	if event.Event == "token.confirmed" {
		h.handleTokenConfirmed(c, body)
		return
	}

	// Handle virtual_account.credited for offline payment reconciliation
	if event.Event == "virtual_account.credited" {
		h.handleVirtualAccountCredited(c, body)
		return
	}

	if event.Event == "payment.failed" {
		h.handleRazorpayPaymentFailed(c, event)
		return
	}

	// Refund lifecycle events: advance the credit note tracking the refund.
	if event.Event == "refund.processed" || event.Event == "refund.failed" {
		h.handleRazorpayRefundEvent(c, body, event.Event == "refund.processed")
		return
	}

	if event.Event == "payment.captured" || event.Event == "order.paid" {
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

		// 2. Mark Invoice as Paid. MarkInvoicePaid reads the invoice through
		// the tenant-scoped repository, and webhook requests carry no tenant —
		// load the invoice first and inject its own tenant id. Already-paid
		// invoices (e.g. mandate debits, which are created paid) no-op inside
		// MarkInvoicePaid, so redelivery is safe.
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
			h.logger.Error("failed to mark invoice paid",
				"invoice_id", invoiceID,
				"error", err,
			)
			respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
			return
		}

		h.logger.Info("invoice marked paid via webhook", "invoice_id", invoiceID)

		// Persist the gateway payment id (pay_*) — refunds are issued against
		// it. For mandate-collected invoices this is the only place the actual
		// payment id becomes known: the debit response returns an order id.
		if paymentID := event.Payload.Payment.Entity.ID; paymentID != "" && h.invoiceRepo != nil {
			if err := h.invoiceRepo.SetGatewayPaymentID(ctx, invoiceID, paymentID); err != nil {
				h.logger.Error("failed to record gateway payment id",
					"invoice_id", invoiceID,
					"payment_id", paymentID,
					"error", err,
				)
			}
		}

		// 3. Record success outcome for RL if this invoice was managed by smart
		// dunning — only when THIS delivery performed the paid transition, so a
		// redelivered webhook (or a second settler) can't double-count the
		// dunning bandit's reward (ENG-162).
		if transitioned {
			h.recordDunningSuccess(ctx, invoiceID)
		}
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleStripe handles incoming Stripe webhook events.
func (h *WebhookHandler) HandleStripe(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read stripe webhook body", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read body")
		return
	}

	// 1. Verify Signature. IgnoreAPIVersionMismatch keeps HMAC verification but
	// tolerates events stamped with a different Stripe API version than the
	// pinned stripe-go release (accounts commonly emit an older default version)
	// — Stripe's recommended handling; without it every delivery 401s.
	var event stripe.Event
	if h.stripeWebhookSecret != "" {
		event, err = webhook.ConstructEventWithOptions(body, c.GetHeader("Stripe-Signature"), h.stripeWebhookSecret,
			webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
		if err != nil {
			h.logger.Warn("stripe webhook signature verification failed",
				"error", err,
				"ip", c.ClientIP(),
			)
			respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid signature")
			return
		}
	} else {
		h.logger.Warn("STRIPE_WEBHOOK_SECRET not set — skipping signature verification")
		if err := json.Unmarshal(body, &event); err != nil {
			h.logger.Error("invalid stripe webhook JSON", "error", err)
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid JSON")
			return
		}
	}

	h.logger.Info("stripe webhook received", "event_type", event.Type)
	ctx := c.Request.Context()

	// Idempotency: skip an event we've already fully processed. Stripe redelivers
	// on non-2xx and can deliver duplicates; without this a redelivery re-runs
	// non-idempotent side effects (the payment-failed email, the dunning bandit
	// outcome). Recorded only on success below, so a failed delivery still
	// retries (ENG-162).
	if h.alreadyProcessed(c, "stripe", event.ID) {
		return
	}

	var handlerErr error
	switch event.Type {
	case "payment_intent.succeeded":
		handlerErr = h.handlePaymentIntentSucceeded(ctx, event)
	case "invoice.payment_failed":
		handlerErr = h.handleInvoicePaymentFailed(ctx, event)
	case "customer.subscription.deleted":
		handlerErr = h.handleSubscriptionDeleted(ctx, event)
	case "charge.refunded":
		handlerErr = h.handleChargeRefunded(ctx, event)
	case "charge.refund.updated", "refund.updated", "refund.failed":
		handlerErr = h.handleStripeRefundUpdated(ctx, event)
	default:
		h.logger.Info("stripe webhook event ignored", "event_type", event.Type)
	}

	if handlerErr != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, handlerErr.Error())
		return
	}

	// Processed cleanly — record it so a redelivery is ignored.
	h.markProcessed(ctx, "stripe", event.ID, string(event.Type))

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

	// MarkInvoicePaid reads the invoice through the tenant-scoped repository,
	// and webhook requests carry no tenant — load the invoice and inject its
	// own tenant id (same as the Razorpay handler).
	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		h.logger.Error("failed to load invoice for stripe payment webhook", "invoice_id", invoiceID, "error", err)
		return fmt.Errorf("failed to load invoice %s: %w", invoiceID, err)
	}
	if inv == nil {
		h.logger.Warn("stripe payment_intent.succeeded ignored — invoice not found", "invoice_id", invoiceID)
		return nil
	}

	ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
	transitioned, err := h.subService.MarkInvoicePaid(ctxWithTenant, invoiceID)
	if err != nil {
		h.logger.Error("failed to mark invoice paid via stripe webhook",
			"invoice_id", invoiceID,
			"error", err,
		)
		return fmt.Errorf("failed to mark invoice paid: %w", err)
	}

	h.logger.Info("invoice marked paid via stripe webhook", "invoice_id", invoiceID)

	// Persist the gateway payment id — refunds are issued against it.
	if h.invoiceRepo != nil && pi.ID != "" {
		if err := h.invoiceRepo.SetGatewayPaymentID(ctx, invoiceID, pi.ID); err != nil {
			h.logger.Error("failed to record gateway payment id",
				"invoice_id", invoiceID,
				"payment_id", pi.ID,
				"error", err,
			)
		}
	}

	// Record the dunning success only when THIS delivery performed the paid
	// transition, so a redelivered payment_intent.succeeded (or a second settler)
	// can't double-count the dunning bandit's reward (ENG-162).
	if transitioned {
		h.recordDunningSuccess(ctx, invoiceID)
	}
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

	// Trigger dunning campaign
	if h.dunningCampaignService != nil {
		if err := h.dunningCampaignService.TriggerCampaign(ctx, invoiceID, "payment_failed"); err != nil {
			h.logger.Error("failed to trigger dunning campaign", "error", err, "invoice_id", invoiceID)
		}
	}

	// Send payment failed notification
	if h.notificationService != nil && h.customerRepo != nil {
		customer, custErr := h.customerRepo.GetByID(ctx, inv.CustomerID)
		if custErr != nil {
			h.logger.Error("failed to fetch customer for payment failed notification", "error", custErr, "customer_id", inv.CustomerID)
		} else if customer != nil {
			failureReason := inv.LastPaymentError
			if failureReason == "" {
				failureReason = "Payment was declined by your bank"
			}
			err := h.notificationService.SendPaymentFailed(ctx, service.PaymentFailedData{
				CustomerName:  stringOrEmpty(customer.Name),
				CustomerEmail: customer.Email,
				InvoiceNumber: inv.InvoiceNumber,
				Amount:        formatInvoiceAmount(inv.Total, inv.Currency),
				FailureReason: failureReason,
			})
			if err != nil {
				h.logger.Error("failed to send payment failed notification", "error", err, "invoice_id", invoiceID)
			}
		}
	}

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
	ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, sub.TenantID)
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

// handleChargeRefunded consumes Stripe charge.refunded. The event carries the
// charge with its refunds; each refund is applied to the credit note that
// owns it (pending → processed / refund_failed).
func (h *WebhookHandler) handleChargeRefunded(ctx context.Context, event stripe.Event) error {
	if h.creditNoteService == nil {
		h.logger.Info("credit note service not configured, ignoring charge.refunded")
		return nil
	}

	var ch stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
		h.logger.Error("failed to unmarshal stripe charge", "error", err)
		return fmt.Errorf("failed to unmarshal stripe charge: %w", err)
	}

	if ch.Refunds == nil || len(ch.Refunds.Data) == 0 {
		h.logger.Info("stripe charge.refunded carried no refund objects", "charge_id", ch.ID)
		return nil
	}

	for _, ref := range ch.Refunds.Data {
		if ref == nil || ref.ID == "" {
			continue
		}
		if err := h.applyStripeRefund(ctx, ref); err != nil {
			return err
		}
	}
	return nil
}

// handleStripeRefundUpdated consumes refund-object events
// (charge.refund.updated, refund.updated, refund.failed) — these are how
// Stripe reports asynchronous refund failures after the initial submission.
func (h *WebhookHandler) handleStripeRefundUpdated(ctx context.Context, event stripe.Event) error {
	if h.creditNoteService == nil {
		h.logger.Info("credit note service not configured, ignoring stripe refund event", "event_type", event.Type)
		return nil
	}

	var ref stripe.Refund
	if err := json.Unmarshal(event.Data.Raw, &ref); err != nil {
		h.logger.Error("failed to unmarshal stripe refund", "error", err)
		return fmt.Errorf("failed to unmarshal stripe refund: %w", err)
	}
	if ref.ID == "" {
		h.logger.Info("stripe refund event carried no refund id", "event_type", event.Type)
		return nil
	}
	return h.applyStripeRefund(ctx, &ref)
}

// applyStripeRefund advances the credit note owning a Stripe refund based on
// the refund's reported status. Refunds still pending at the gateway are left
// alone; refund ids that match no credit note are logged and swallowed so the
// webhook is acknowledged (Stripe retries non-2xx responses indefinitely).
func (h *WebhookHandler) applyStripeRefund(ctx context.Context, ref *stripe.Refund) error {
	var succeeded bool
	reason := ""
	switch ref.Status {
	case stripe.RefundStatusSucceeded:
		succeeded = true
	case stripe.RefundStatusFailed, stripe.RefundStatusCanceled:
		reason = string(ref.FailureReason)
		if reason == "" {
			reason = fmt.Sprintf("stripe reported refund status %q", ref.Status)
		}
	default:
		// pending / requires_action — nothing to advance yet; a later event
		// will settle it.
		h.logger.Info("stripe refund not settled yet, leaving credit note pending",
			"refund_id", ref.ID, "refund_status", ref.Status)
		return nil
	}

	err := h.creditNoteService.ProcessGatewayRefundEvent(ctx, ref.ID, succeeded, reason)
	if errors.Is(err, service.ErrRefundNotFound) {
		h.logger.Info("stripe refund event ignored — no matching credit note", "refund_id", ref.ID)
		return nil
	}
	if err != nil {
		h.logger.Error("failed to process stripe refund event", "refund_id", ref.ID, "error", err)
	}
	return err
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
	// Mark dunning campaign as recovered
	if h.dunningCampaignService != nil {
		if err := h.dunningCampaignService.MarkRecovered(ctx, invoiceID); err != nil {
			h.logger.Error("failed to mark dunning campaign recovered", "error", err, "invoice_id", invoiceID)
		}
	}

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

func formatInvoiceAmount(amountPaise int64, currency string) string {
	amount := float64(amountPaise) / 100
	switch currency {
	case "INR":
		return fmt.Sprintf("₹%.2f", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%s %.2f", currency, amount)
	}
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
