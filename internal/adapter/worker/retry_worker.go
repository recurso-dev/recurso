package worker

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

type RetryWorker struct {
	invoiceRepo  port.InvoiceRepository
	retryService *service.SmartRetryService
	gateway      port.PaymentGateway
	notifier     port.Notifier
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

	// 1. Attempt payment via gateway
	result, err := w.gateway.RetryPayment(ctx, inv.ID.String(), inv.Total, inv.Currency)
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
		// Mark invoice paid
		log.Printf("Worker: Payment succeeded for %s (payment_id=%s)", inv.InvoiceNumber, result.PaymentID)
		now := time.Now()
		inv.Status = domain.InvoiceStatusPaid
		inv.PaidAt = &now
		inv.NextRetryAt = nil
		inv.DunningActionID = ""
		inv.DunningContextKey = ""
		inv.LastPaymentError = ""
		if updateErr := w.invoiceRepo.Update(ctx, inv); updateErr != nil {
			log.Printf("Worker: Failed to mark invoice %s as paid: %v", inv.ID, updateErr)
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
