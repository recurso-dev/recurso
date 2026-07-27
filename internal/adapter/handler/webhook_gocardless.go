package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// gcEvent is one entry in a GoCardless webhook payload. Unlike Razorpay and
// Stripe, GoCardless batches many events per delivery, so processing (and
// dedup) is per event, never per request.
type gcEvent struct {
	ID           string `json:"id"`
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	Links        struct {
		Mandate               string `json:"mandate"`
		BillingRequest        string `json:"billing_request"`
		MandateRequestMandate string `json:"mandate_request_mandate"`
		Payment               string `json:"payment"`
	} `json:"links"`
}

// HandleGoCardless processes GoCardless webhook deliveries: billing-request
// fulfilment activates the local mandate row (and swaps the stored BRQ... id
// for the real MD... mandate id future debits reference); mandate lifecycle
// events keep the row's status in sync. Everything else is acknowledged and
// ignored. Registered at /webhooks/gocardless (platform secret) and
// /webhooks/gocardless/:connID (per-BYO-connection secret).
func (h *WebhookHandler) HandleGoCardless(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "failed to read request body")
		return
	}

	envSecret := os.Getenv("GOCARDLESS_WEBHOOK_SECRET")
	if envSecret == "" && c.Param("connID") == "" {
		h.logger.Error("GOCARDLESS_WEBHOOK_SECRET not set — rejecting webhook (fail closed)", "ip", c.ClientIP())
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "webhook verification not configured")
		return
	}
	secret, ok := h.webhookSecretFor(c, domain.GatewayGoCardless, envSecret)
	if !ok {
		return
	}

	// GoCardless signs the raw body with HMAC-SHA256 (hex) in Webhook-Signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(c.GetHeader("Webhook-Signature"))) {
		h.logger.Error("invalid GoCardless webhook signature", "ip", c.ClientIP())
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "invalid webhook signature")
		return
	}

	var payload struct {
		Events []gcEvent `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid webhook payload")
		return
	}

	if h.mandateService == nil {
		h.logger.Info("mandate service not configured, ignoring GoCardless events")
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	processed := 0
	for _, ev := range payload.Events {
		if h.gcEventProcessed(c, ev.ID) {
			continue
		}
		if err := h.handleGoCardlessEvent(c, ev); err != nil {
			// Log and continue: one unmatched event must not block the batch,
			// and GoCardless retries whole deliveries (dedup skips the done ones).
			h.logger.Error("failed to handle GoCardless event",
				"event_id", ev.ID, "resource", ev.ResourceType, "action", ev.Action, "error", err)
			continue
		}
		h.markProcessed(c.Request.Context(), "gocardless", ev.ID, ev.ResourceType+"."+ev.Action)
		processed++
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "processed": processed})
}

func (h *WebhookHandler) handleGoCardlessEvent(c *gin.Context, ev gcEvent) error {
	ctx := c.Request.Context()
	switch ev.ResourceType {
	case "billing_requests":
		if ev.Action != "fulfilled" {
			return nil
		}
		if err := h.mandateService.HandleGoCardlessFulfilment(ctx, ev.Links.BillingRequest, ev.Links.MandateRequestMandate); err != nil {
			return err
		}
		h.logger.Info("GoCardless mandate activated via billing request",
			"billing_request", ev.Links.BillingRequest, "gc_mandate", ev.Links.MandateRequestMandate)
	case "mandates":
		if err := h.mandateService.HandleGoCardlessMandateEvent(ctx, ev.Links.Mandate, ev.Action); err != nil {
			return err
		}
		h.logger.Info("GoCardless mandate event applied", "gc_mandate", ev.Links.Mandate, "action", ev.Action)
	case "payments":
		return h.handleGoCardlessPaymentEvent(c, ev)
	}
	return nil
}

// invoiceByGatewayPayment is the optional repo capability the GoCardless
// settlement path needs; satisfied by *db.InvoiceRepository. Capability
// assertion instead of widening port.InvoiceRepository, so existing mocks
// keep compiling.
type invoiceByGatewayPayment interface {
	GetByGatewayPaymentIDPublic(ctx context.Context, gatewayPaymentID string) (*domain.Invoice, error)
}

// handleGoCardlessPaymentEvent settles or flags an invoice on the outcome of
// an asynchronous bank debit. ExecuteMandateDebit leaves the invoice OPEN and
// stores the GoCardless payment id (PM...) on it; "confirmed" is the payoff.
func (h *WebhookHandler) handleGoCardlessPaymentEvent(c *gin.Context, ev gcEvent) error {
	ctx := c.Request.Context()
	switch ev.Action {
	case "confirmed", "paid_out":
		repo, ok := h.invoiceRepo.(invoiceByGatewayPayment)
		if !ok {
			h.logger.Info("invoice repo lacks gateway-payment lookup, ignoring GoCardless payment event", "payment", ev.Links.Payment)
			return nil
		}
		inv, err := repo.GetByGatewayPaymentIDPublic(ctx, ev.Links.Payment)
		if err != nil {
			return err // transient lookup failure: leave unprocessed so GoCardless redelivers
		}
		if inv == nil {
			h.logger.Info("no invoice references GoCardless payment, ignoring", "payment", ev.Links.Payment)
			return nil
		}
		if h.subService == nil {
			h.logger.Info("subscription service not configured, ignoring GoCardless settlement", "payment", ev.Links.Payment)
			return nil
		}
		ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
		transitioned, err := h.subService.MarkInvoicePaid(ctxWithTenant, inv.ID)
		if err != nil {
			return err
		}
		h.logger.Info("invoice settled via GoCardless payment event",
			"invoice_id", inv.ID, "payment", ev.Links.Payment, "action", ev.Action)
		if transitioned {
			h.recordDunningSuccess(ctx, inv.ID)
		}
	case "failed", "cancelled":
		// The debit never stuck; its invoice was left OPEN awaiting settlement,
		// so dunning picks it up — no state change needed here.
		h.logger.Warn("GoCardless debit did not settle; invoice remains open for dunning",
			"payment", ev.Links.Payment, "action", ev.Action)
	case "charged_back", "late_failure":
		// The SEPA/Bacs equivalent of an ACH late return: money that already
		// settled is pulled back by the bank. Unlike Stripe (where returns
		// arrive dressed as refunds and need classification guards, #210),
		// GoCardless models merchant refunds as a separate resource — a
		// payments.charged_back/late_failure is unambiguously involuntary.
		repo, ok := h.invoiceRepo.(invoiceByGatewayPayment)
		if !ok || h.subService == nil {
			h.logger.Error("GoCardless payment reversed after settlement but reversal path not configured — BOOKS NEED REVIEW",
				"payment", ev.Links.Payment, "action", ev.Action)
			return nil
		}
		inv, err := repo.GetByGatewayPaymentIDPublic(ctx, ev.Links.Payment)
		if err != nil {
			return err // transient: leave unprocessed so GoCardless redelivers
		}
		if inv == nil {
			h.logger.Warn("GoCardless payment reversed but no invoice references it — ignoring",
				"payment", ev.Links.Payment, "action", ev.Action)
			return nil
		}
		// ReverseSettledPayment is idempotent on the paid guard: it reopens
		// (paid -> past_due) and posts the inverse cash leg (code 19,
		// occurrence-aware) only on the transitioning delivery. The reopened
		// invoice re-enters dunning on its normal cadence.
		ctxWithTenant := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
		reversed, err := h.subService.ReverseSettledPayment(ctxWithTenant, inv.ID)
		if err != nil {
			return err
		}
		h.logger.Warn("GoCardless payment clawed back — settlement reversed",
			"payment", ev.Links.Payment, "action", ev.Action, "invoice_id", inv.ID, "reversed", reversed)
	}
	return nil
}

// gcEventProcessed is alreadyProcessed without the per-request acknowledgement
// response — GoCardless batches events, so a duplicate is skipped silently
// while the rest of the batch still runs.
func (h *WebhookHandler) gcEventProcessed(c *gin.Context, eventID string) bool {
	if h.inboundDedup == nil || eventID == "" {
		return false
	}
	done, err := h.inboundDedup.WasProcessed(c.Request.Context(), "gocardless", eventID)
	if err != nil {
		h.logger.Error("webhook dedup check failed; processing anyway", "gateway", "gocardless", "event_id", eventID, "error", err)
		return false
	}
	return done
}
