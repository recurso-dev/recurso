package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

// HandleStripe handles incoming Stripe webhook events.
func (h *WebhookHandler) HandleStripe(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read stripe webhook body", "error", err)
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read body")
		return
	}

	// 1. Verify Signature. Fail CLOSED: an unconfigured secret must REJECT the
	// webhook, never process it unverified — otherwise a forged
	// invoice.payment_succeeded / checkout.session.completed with a known
	// invoice_id would settle an invoice with no real payment on a misconfigured
	// deploy. (Razorpay fails closed the same way; this path previously fell open,
	// only logging a warning — ENG-175.)
	if h.stripeWebhookSecret == "" && c.Param("connID") == "" {
		h.logger.Error("STRIPE_WEBHOOK_SECRET not set — rejecting webhook (fail closed)", "ip", c.ClientIP())
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "webhook verification not configured")
		return
	}
	// Per-connection route (:connID) verifies with that tenant's own webhook
	// secret; the legacy route uses the env secret.
	stripeSecret, ok := h.webhookSecretFor(c, domain.GatewayStripe, h.stripeWebhookSecret)
	if !ok {
		return
	}
	// IgnoreAPIVersionMismatch keeps HMAC verification but tolerates events
	// stamped with a different Stripe API version than the pinned stripe-go
	// release (accounts commonly emit an older default version) — Stripe's
	// recommended handling; without it every delivery 401s.
	event, err := webhook.ConstructEventWithOptions(body, c.GetHeader("Stripe-Signature"), stripeSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		h.logger.Warn("stripe webhook signature verification failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid signature")
		return
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
		respondInternalError(c, handlerErr)
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
		if err := h.invoiceRepo.SetGatewayPaymentID(ctx, inv.TenantID, invoiceID, pi.ID); err != nil {
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
		if errors.Is(err, sql.ErrNoRows) {
			// No local subscription maps to this Stripe id — it was created
			// outside Recurso, already removed, or never synced. There is nothing
			// to cancel, and retrying can't change that, so ACK (return nil → 200)
			// instead of erroring: a 500 here makes Stripe redeliver the same
			// deletion indefinitely.
			h.logger.Info("stripe subscription.deleted for unknown subscription; acking",
				"stripe_subscription_id", stripeSub.ID)
			return nil
		}
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
