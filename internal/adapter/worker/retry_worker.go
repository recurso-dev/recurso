package worker

import (
	"context"
	"fmt"
	"log"
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
	MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) error
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

	log.Println("RetryWorker started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("RetryWorker stopping...")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *RetryWorker) processRetries(ctx context.Context) {
	invoices, err := w.invoiceRepo.GetDueForRetry(ctx)
	if err != nil {
		log.Printf("Worker: Failed to fetch retry invoices: %v", err)
		return
	}

	if len(invoices) > 0 {
		log.Printf("Worker: Found %d invoices to retry", len(invoices))
	}

	for _, inv := range invoices {
		w.processInvoice(ctx, inv)
	}
}

func (w *RetryWorker) processInvoice(ctx context.Context, inv *domain.Invoice) {
	log.Printf("Worker: Retrying Invoice %s (Attempt %d)", inv.InvoiceNumber, inv.RetryCount+1)

	// 1. Attempt payment — via the customer's saved method (off-session) when
	// available, else the interactive gateway path.
	result, err := w.chargeInvoice(ctx, inv)
	if err != nil {
		// Infrastructure error (network, etc.) — schedule short retry, don't record outcome
		log.Printf("Worker: Gateway infra error for %s: %v — scheduling 5min retry", inv.InvoiceNumber, err)
		shortRetry := time.Now().Add(5 * time.Minute)
		inv.NextRetryAt = &shortRetry
		if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
			log.Printf("Worker: Failed to update invoice %s: %v", inv.ID, updateErr)
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
			log.Printf("Worker: Failed to record outcome for %s: %v", inv.InvoiceNumber, historyErr)
		} else {
			log.Printf("Worker: Recorded outcome=%s for action=%s context=%s", outcome, inv.DunningActionID, inv.DunningContextKey)
		}
	}

	// 3. Handle result
	if result.Success {
		log.Printf("Worker: Payment succeeded for %s (payment_id=%s)", inv.InvoiceNumber, result.PaymentID)
		if w.settler != nil {
			// Settle through the ledger path — the same idempotent method the
			// checkout and payment webhooks use, so amount_paid, recovery
			// attribution, the ledger entry, and the rev-rec schedule all
			// happen here rather than in a racing webhook that would no-op on
			// an already-paid invoice. The worker's ctx carries no tenant, so
			// inject the invoice's own (same as the webhook handlers).
			tenantCtx := context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)
			if err := w.settler.MarkInvoicePaid(tenantCtx, inv.ID); err != nil {
				// Leave dunning state untouched: the next poll retries and the
				// settle is idempotent, so nothing is lost or double-posted.
				log.Printf("Worker: Failed to settle invoice %s: %v", inv.ID, err)
				return
			}
			// Re-read the settled row before clearing dunning state — Update
			// writes the full struct and would clobber paid_at/amount_paid.
			settled, err := w.invoiceRepo.GetByID(tenantCtx, inv.ID)
			if err != nil || settled == nil {
				log.Printf("Worker: Failed to reload settled invoice %s: %v", inv.ID, err)
				return
			}
			settled.NextRetryAt = nil
			settled.DunningActionID = ""
			settled.DunningContextKey = ""
			settled.LastPaymentError = ""
			if updateErr := w.invoiceRepo.Update(ctx, settled); updateErr != nil {
				log.Printf("Worker: Failed to clear dunning state for %s: %v", inv.ID, updateErr)
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
				log.Printf("Worker: Failed to mark invoice %s as paid: %v", inv.ID, updateErr)
			} else if w.recoveryRecorder != nil {
				// Recovered revenue attribution (idempotent, non-fatal)
				w.recoveryRecorder.RecordIfRecovered(ctx, &recoverySnapshot)
			}
		}

		// Mark dunning campaign as recovered
		if w.dunningCampaignService != nil {
			if err := w.dunningCampaignService.MarkRecovered(ctx, inv.ID); err != nil {
				log.Printf("Worker: Failed to mark dunning campaign recovered for %s: %v", inv.ID, err)
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
		log.Printf("Worker: Max retries reached for %s. Marking uncollectible.", inv.InvoiceNumber)
		inv.Status = domain.InvoiceStatusUncollectible
		inv.NextRetryAt = nil
		inv.DunningActionID = ""
		inv.DunningContextKey = ""
		if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
			log.Printf("Worker: Failed to mark invoice %s uncollectible: %v", inv.ID, updateErr)
		}
		return
	}

	// 5. Schedule next retry with RL decision tracking
	log.Printf("Worker: Payment failed (%s). Next retry: action=%s at %v", errorCode, decision.Action.ID, decision.NextRetryAt)
	inv.NextRetryAt = &decision.NextRetryAt
	inv.Status = domain.InvoiceStatusPastDue
	inv.DunningActionID = decision.Action.ID
	inv.DunningContextKey = decision.ContextKey

	if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
		log.Printf("Worker: Failed to update invoice %s: %v", inv.ID, updateErr)
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
