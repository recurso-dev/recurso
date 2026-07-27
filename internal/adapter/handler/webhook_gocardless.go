package handler

import (
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
