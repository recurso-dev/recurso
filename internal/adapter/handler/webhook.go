package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
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
	paymentAttempts        paymentAttemptStore // nil-safe; ACH async settlement state (Inc 3b)
	stripeWebhookSecret    string
	gatewayConns           gatewayConnResolver // nil-safe; per-connection (BYO) webhook secrets
	logger                 *slog.Logger
}

// gatewayConnResolver resolves a BYO connection by id and decrypts its webhook
// signing secret. Satisfied by *service.GatewayConnectionService. Used only by
// the per-connection webhook routes (/webhooks/{stripe,razorpay}/:connID); the
// legacy env routes never touch it.
type gatewayConnResolver interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.GatewayConnection, error)
	OpenWebhookSecret(conn *domain.GatewayConnection) (string, error)
}

// SetGatewayConnections wires per-connection webhook secret resolution (BYO
// increment 3). Nil-safe: unset, only the env webhook routes work.
func (h *WebhookHandler) SetGatewayConnections(r gatewayConnResolver) { h.gatewayConns = r }

// webhookConnTenantKey holds, in the request context, the tenant of the BYO
// connection a per-connection webhook authenticated as (set by webhookSecretFor).
// Absent on the legacy env route, where the platform holds the secret.
type webhookConnTenantKeyType struct{}

var webhookConnTenantKey webhookConnTenantKeyType

// webhookConnTenant returns the BYO connection's tenant for this request, or
// uuid.Nil on the legacy env route.
func webhookConnTenant(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(webhookConnTenantKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// invoiceBelongsToWebhookConn reports whether it is safe for this webhook to act
// on inv. On a BYO (:connID) route it requires inv to belong to the connection's
// own tenant, so a signed payload can't reference another tenant's invoice. On
// the legacy env route (no conn tenant) it is always true. A nil invoice is
// handled by the caller's own not-found path.
func invoiceBelongsToWebhookConn(ctx context.Context, inv *domain.Invoice) bool {
	ct := webhookConnTenant(ctx)
	if ct == uuid.Nil || inv == nil {
		return true
	}
	return inv.TenantID == ct
}

// webhookSecretFor resolves the signing secret to verify an inbound webhook.
// With a :connID path param it looks up that BYO connection and decrypts its
// webhook secret (fail closed on any problem); without one it returns the env
// secret. The bool is false when it has already written an error response.
func (h *WebhookHandler) webhookSecretFor(c *gin.Context, provider domain.GatewayProvider, envSecret string) (string, bool) {
	connID := c.Param("connID")
	if connID == "" {
		return envSecret, true // legacy platform-gateway route
	}
	if h.gatewayConns == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "per-connection webhooks not configured")
		return "", false
	}
	id, err := uuid.Parse(connID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid connection id")
		return "", false
	}
	conn, err := h.gatewayConns.GetByID(c.Request.Context(), id)
	// Do not leak which ids exist: any resolution failure is a flat 404.
	if err != nil || conn == nil || conn.Provider != provider || !conn.Active {
		respondError(c, http.StatusNotFound, codeNotFound, "connection not found")
		return "", false
	}
	// Bind this request to the connection's tenant. The signature is verified with
	// THIS connection's secret, but the invoice a handler acts on is loaded by the
	// payload's id (unscoped) — so every side effect must confirm the target
	// belongs to this tenant, or a BYO tenant could settle/refund another tenant's
	// invoice by signing a body carrying its id (ENG-172). Stored on the request
	// context so both the gin-context handlers (Razorpay) and the ctx-only
	// handlers (Stripe) can read it via webhookConnTenant / invoiceBelongsToWebhookConn.
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), webhookConnTenantKey, conn.TenantID))
	secret, err := h.gatewayConns.OpenWebhookSecret(conn)
	if err != nil || secret == "" {
		// Fail closed: a connection without a webhook secret cannot be verified.
		h.logger.Error("BYO connection has no webhook secret — rejecting (fail closed)",
			"connection_id", id, "provider", provider, "ip", c.ClientIP())
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "webhook verification not configured for this connection")
		return "", false
	}
	return secret, true
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

// SetPaymentAttempts wires the ACH async-settlement store so the Stripe webhook
// records processing/succeeded/failed transitions (Inc 3b). Nil-safe.
func (h *WebhookHandler) SetPaymentAttempts(s paymentAttemptStore) { h.paymentAttempts = s }

// alreadyProcessed acknowledges and returns true when this (gateway, eventID)
// was already fully processed OR the dedup store could not be consulted — in
// both cases the caller must stop (a response has been written). Fails open
// (returns false) only for a nil store or empty id — configuration choices,
// not transient faults. A LOOKUP ERROR fails CLOSED with a 503 (S4): every
// gateway retries on a non-2xx, so a store outage merely defers the event
// until the store recovers — whereas processing on faith risks re-running
// non-idempotent side effects (payment-failed emails, dunning bandit outcomes)
// on a redelivery the broken dedup could not recognize.
func (h *WebhookHandler) alreadyProcessed(c *gin.Context, gateway, eventID string) bool {
	if h.inboundDedup == nil || eventID == "" {
		return false
	}
	processed, err := h.inboundDedup.WasProcessed(c.Request.Context(), gateway, eventID)
	if err != nil {
		h.logger.Error("webhook dedup check failed; deferring to gateway retry", "gateway", gateway, "event_id", eventID, "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dedup store unavailable; retry"})
		return true
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

// verifyRazorpaySignature verifies the HMAC SHA256 signature from Razorpay webhooks
// isGatewayPaymentID reports whether id is a real gateway PAYMENT id (Razorpay
// pay_*, Stripe pi_*/ch_*) rather than an order id. Only these are refundable,
// so only these may be written to invoices.gateway_payment_id.
func isGatewayPaymentID(id string) bool {
	return strings.HasPrefix(id, "pay_") || strings.HasPrefix(id, "pi_") || strings.HasPrefix(id, "ch_")
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
	// Exponent-aware: hardcoding /100 misstated non-2-decimal currencies.
	return domain.FormatMoney(amountPaise, currency)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
