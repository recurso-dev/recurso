package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// RecoveryRecorder records recovered-revenue attribution when a retried
// payment succeeds. Implemented by service.DunningRecoveryService.
type RecoveryRecorder interface {
	RecordIfRecovered(ctx context.Context, inv *domain.Invoice) bool
}

// savedMethodCharger charges a customer's saved payment method off-session
// (ENG-5 Phase 2). Implemented by *gateway.StripeGateway; wired only when
// Stripe is configured.
type savedMethodCharger interface {
	ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
}

// customerPaymentLookup resolves a customer's saved payment method. Implemented
// by *db.CustomerRepository.
type customerPaymentLookup interface {
	GetSavedPaymentMethod(ctx context.Context, customerID uuid.UUID) (stripeCustomerID, paymentMethodID string, err error)
}

// invoiceSettler marks an invoice paid through the ledger path (amount_paid,
// recovery attribution, ledger entry, rev-rec schedule) — the same idempotent
// method the checkout and payment webhooks use. Implemented by
// *service.SubscriptionService.
type invoiceSettler interface {
	MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}

type RetryWorker struct {
	invoiceRepo            port.InvoiceRepository
	retryService           *service.SmartRetryService
	gateway                port.PaymentGateway
	notifier               port.Notifier
	dunningCampaignService *service.DunningCampaignService
	recoveryRecorder       RecoveryRecorder
	savedCharger           savedMethodCharger
	customerLookup         customerPaymentLookup
	settler                invoiceSettler
}

func NewRetryWorker(
	invoiceRepo port.InvoiceRepository,
	retryService *service.SmartRetryService,
	gateway port.PaymentGateway,
	notifier port.Notifier,
) *RetryWorker {
	return &RetryWorker{
		invoiceRepo:  invoiceRepo,
		retryService: retryService,
		gateway:      gateway,
		notifier:     notifier,
	}
}

func (w *RetryWorker) SetDunningCampaignService(svc *service.DunningCampaignService) {
	w.dunningCampaignService = svc
}

func (w *RetryWorker) SetRecoveryRecorder(rr RecoveryRecorder) {
	w.recoveryRecorder = rr
}

// SetSettler routes successful retries through the ledger-posting
// MarkInvoicePaid instead of a bare status update, so recovered payments get
// amount_paid, a ledger entry, and a rev-rec schedule like every other payment.
func (w *RetryWorker) SetSettler(s invoiceSettler) {
	w.settler = s
}

// SetSavedMethodCharging wires off-session collection with the customer's saved
// card (ENG-5 Phase 2). When either dependency is nil, retries fall back to the
// interactive gateway path — behavior is unchanged for customers without a
// saved payment method.
func (w *RetryWorker) SetSavedMethodCharging(charger savedMethodCharger, lookup customerPaymentLookup) {
	w.savedCharger = charger
	w.customerLookup = lookup
}

// chargeInvoice attempts collection, preferring the customer's saved payment
// method (charged off-session) when one exists, and falling back to the
// interactive gateway retry otherwise.
func (w *RetryWorker) chargeInvoice(ctx context.Context, inv *domain.Invoice) (*port.PaymentResult, error) {
	if w.savedCharger != nil && w.customerLookup != nil {
		stripeCustomerID, paymentMethodID, err := w.customerLookup.GetSavedPaymentMethod(ctx, inv.CustomerID)
		if err == nil && stripeCustomerID != "" && paymentMethodID != "" {
			// Idempotent per attempt: a worker re-run for the same attempt won't
			// double-charge.
			key := fmt.Sprintf("retry-%s-%d", inv.ID, inv.RetryCount)
			return w.savedCharger.ChargeSavedPaymentMethod(ctx, stripeCustomerID, paymentMethodID, inv.Total, inv.Currency, inv.ID.String(), key)
		}
	}
	return w.gateway.RetryPayment(ctx, inv.ID.String(), inv.Total, inv.Currency)
}

// Start runs the worker loop.
func (w *RetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Poll every 10s for demo
	defer ticker.Stop()

	slog.Info("retry worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("retry worker stopping")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *RetryWorker) processRetries(ctx context.Context) {
	invoices, err := w.invoiceRepo.GetDueForRetry(ctx)
	if err != nil {
		slog.Error("retry worker: failed to fetch retry invoices", "error", err)
		return
	}

	if len(invoices) > 0 {
		slog.Info("retry worker: found invoices to retry", "count", len(invoices))
	}

	for _, inv := range invoices {
		w.processInvoice(ctx, inv)
	}
}

func (w *RetryWorker) processInvoice(ctx context.Context, inv *domain.Invoice) {
	slog.Info("retry worker: retrying invoice", "invoice_number", inv.InvoiceNumber, "attempt", inv.RetryCount+1)

	// 1. Attempt payment — via the customer's saved method (off-session) when
	// available, else the interactive gateway path.
	result, err := w.chargeInvoice(ctx, inv)
	if err != nil {
		// Infrastructure error (network, etc.) — schedule short retry, don't record outcome
		slog.Warn("retry worker: gateway infra error, scheduling 5min retry", "invoice_number", inv.InvoiceNumber, "error", err)
		shortRetry := time.Now().Add(5 * time.Minute)
		inv.NextRetryAt = &shortRetry
		if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
			slog.Error("retry worker: failed to update invoice", "invoice_id", inv.ID, "error", updateErr)
		}
		return
	}

	// 2. Record outcome for the PREVIOUS action (if one was tracked)
	if inv.DunningActionID != "" && inv.DunningContextKey != "" {
		outcome := "failure"
		reward := 0.0
		if result.Success {
			outcome = "success"
			reward = 1.0
		}

		historyErr := w.retryService.RecordOutcome(ctx, domain.DunningHistory{
			ID:            uuid.New(),
			TenantID:      inv.TenantID,
			InvoiceID:     inv.ID,
			ContextKey:    inv.DunningContextKey,
			ActionID:      inv.DunningActionID,
			RetryInterval: getDurationSeconds(inv.DunningActionID),
			Outcome:       outcome,
			Reward:        reward,
			CreatedAt:     time.Now(),
		})
		if historyErr != nil {
			slog.Error("retry worker: failed to record outcome", "invoice_number", inv.InvoiceNumber, "error", historyErr)
		} else {
			slog.Info("retry worker: recorded outcome", "outcome", outcome, "action_id", inv.DunningActionID, "context_key", inv.DunningContextKey)
		}
	}

	// 3. Handle result
	if result.Success {
		slog.Info("retry worker: payment succeeded", "invoice_number", inv.InvoiceNumber, "payment_id", result.PaymentID)
		if w.settler != nil {
			// Settle through the ledger path — the same idempotent method the
			// checkout and payment webhooks use, so amount_paid, recovery
			// attribution, the ledger entry, and the rev-rec schedule all
			// happen here rather than in a racing webhook that would no-op on
			// an already-paid invoice. The worker's ctx carries no tenant, so
			// inject the invoice's own (same as the webhook handlers).
			tenantCtx := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
			if _, err := w.settler.MarkInvoicePaid(tenantCtx, inv.ID); err != nil {
				// Leave dunning state untouched: the next poll retries and the
				// settle is idempotent, so nothing is lost or double-posted.
				slog.Error("retry worker: failed to settle invoice", "invoice_id", inv.ID, "error", err)
				return
			}
			// Re-read the settled row before clearing dunning state — Update
			// writes the full struct and would clobber paid_at/amount_paid.
			settled, err := w.invoiceRepo.GetByID(tenantCtx, inv.ID)
			if err != nil || settled == nil {
				slog.Error("retry worker: failed to reload settled invoice", "invoice_id", inv.ID, "error", err)
				return
			}
			settled.NextRetryAt = nil
			settled.DunningActionID = ""
			settled.DunningContextKey = ""
			settled.LastPaymentError = ""
			if updateErr := w.invoiceRepo.Update(ctx, settled); updateErr != nil {
				slog.Error("retry worker: failed to clear dunning state", "invoice_id", inv.ID, "error", updateErr)
			}
		} else {
			// No settler wired (tests / minimal setups): the legacy direct
			// update, which skips the ledger and rev-rec.
			now := time.Now()
			inv.Status = domain.InvoiceStatusPaid
			inv.PaidAt = &now
			inv.AmountPaid = inv.Total
			// Snapshot before dunning fields are cleared — recovery attribution
			// needs retry_count and the action that was in effect at payment time.
			recoverySnapshot := *inv
			inv.NextRetryAt = nil
			inv.DunningActionID = ""
			inv.DunningContextKey = ""
			inv.LastPaymentError = ""
			if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
				slog.Error("retry worker: failed to mark invoice as paid", "invoice_id", inv.ID, "error", updateErr)
			} else if w.recoveryRecorder != nil {
				// Recovered revenue attribution (idempotent, non-fatal)
				w.recoveryRecorder.RecordIfRecovered(ctx, &recoverySnapshot)
			}
		}

		// Mark dunning campaign as recovered
		if w.dunningCampaignService != nil {
			if err := w.dunningCampaignService.MarkRecovered(ctx, inv.ID); err != nil {
				slog.Error("retry worker: failed to mark dunning campaign recovered", "invoice_id", inv.ID, "error", err)
			}
		}
		return
	}

	// 4. Payment failed — decide next retry
	inv.RetryCount++
	errorCode := result.ErrorCode
	if errorCode == "" {
		errorCode = "GENERIC_FAILURE"
	}
	inv.LastPaymentError = errorCode

	decision := w.retryService.DecideRetry(ctx, inv, errorCode)
	if decision == nil {
		// Max retries reached
		slog.Warn("retry worker: max retries reached, marking uncollectible", "invoice_number", inv.InvoiceNumber)
		inv.Status = domain.InvoiceStatusUncollectible
		inv.NextRetryAt = nil
		inv.DunningActionID = ""
		inv.DunningContextKey = ""
		if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
			slog.Error("retry worker: failed to mark invoice uncollectible", "invoice_id", inv.ID, "error", updateErr)
		}
		return
	}

	// 5. Schedule next retry with RL decision tracking
	slog.Warn("retry worker: payment failed, scheduling next retry", "error_code", errorCode, "action_id", decision.Action.ID, "next_retry_at", decision.NextRetryAt)
	inv.NextRetryAt = &decision.NextRetryAt
	inv.Status = domain.InvoiceStatusPastDue
	inv.DunningActionID = decision.Action.ID
	inv.DunningContextKey = decision.ContextKey

	if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
		slog.Error("retry worker: failed to update invoice", "invoice_id", inv.ID, "error", updateErr)
	}
}

// getDurationSeconds maps action ID to its interval in seconds
func getDurationSeconds(actionID string) int64 {
	for _, a := range domain.DefaultDunningActions {
		if a.ID == actionID {
			return int64(a.Interval.Seconds())
		}
	}
	return 86400 // default 24h
}
