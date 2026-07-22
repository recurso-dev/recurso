package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ConvertTrialToActive converts a trialing subscription to active and generates
// its first real invoice. The invoice is created "open" with a due date, so it
// enters the normal payment/dunning path (GetOverdueInvoices picks up open
// invoices once past due). Returns an error if the subscription is not trialing.
func (s *SubscriptionService) ConvertTrialToActive(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	if sub == nil {
		return nil, fmt.Errorf("subscription is nil")
	}
	if sub.Status != domain.SubscriptionStatusTrialing {
		return nil, fmt.Errorf("subscription %s is not trialing (status=%s)", sub.ID, sub.Status)
	}

	// The trial scheduler calls with a background context, but every repo
	// below is tenant-scoped — inject the subscription's own tenant (same
	// pattern as the payment webhooks). Without this no trial ever converts.
	ctx = context.WithValue(ctx, domain.TenantIDKey, sub.TenantID)

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil || len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}

	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	price := plan.Prices[0]
	now := time.Now().UTC()

	// The first paid period starts where the trial ended.
	start := now
	if sub.TrialEnd != nil {
		start = *sub.TrialEnd
	}
	end := domain.AddInterval(start, string(plan.IntervalUnit), plan.IntervalCount)

	subtotal := price.Amount
	taxRes := s.taxResolver.ResolveInvoiceTax(ctx, sub.TenantID, customer, price.Currency, subtotal, plan.HSNCode)
	total := subtotal + taxRes.Total

	paymentTerms := sub.PaymentTerms
	if paymentTerms == "" {
		paymentTerms = "due_on_receipt"
	}
	dueDate := domain.CalculateDueDate(now, paymentTerms)

	invID := uuid.New()
	convDesc := plan.Name
	if convDesc == "" {
		convDesc = "Subscription"
	}
	invoice := &domain.Invoice{
		ID:             invID,
		TenantID:       sub.TenantID,
		EntityID:       sub.EntityID,
		SubscriptionID: &sub.ID,
		CustomerID:     sub.CustomerID,
		BillingReason:  domain.BillingReasonSubscriptionCycle,
		Status:         domain.InvoiceStatusOpen,
		Currency:       price.Currency,
		Subtotal:       subtotal,
		TaxAmount:      taxRes.Total,
		TaxType:        taxRes.TaxType, // D3c: persist for the liability report
		Total:          total,
		IGSTAmount:     taxRes.IGST,
		CGSTAmount:     taxRes.CGST,
		SGSTAmount:     taxRes.SGST,
		// Itemization (Phase 1): single plan line reconciling to the totals.
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(invID, convDesc, taxRes.HSN, 1, subtotal, subtotal, taxRes, time.Time{}),
		},
		PaymentTerms: paymentTerms,
		CreatedAt:    now,
		DueDate:      dueDate,
	}

	// P25 e-invoicing is deferred to AFTER the invoice + activation commit (below),
	// so a rolled-back conversion — including the atomic trial-race loss — can't
	// orphan an irreversible government IRN (PHASE2 #3).

	sub.Status = domain.SubscriptionStatusActive
	sub.CurrentPeriodStart = start
	sub.CurrentPeriodEnd = end
	sub.UpdatedAt = now

	// Persist the first invoice and activate the subscription atomically, so a
	// mid-write failure can't leave a trial billed but still trialing (or
	// activated with no invoice) (ENG-150). The activation is a CONDITIONAL
	// transition (WHERE status='trialing'): with the scheduler lock a no-op under
	// multi-instance, two runners can both read this trial as expired, but only
	// the one that flips the row creates an invoice — the loser matches zero rows
	// and bails out, so a trial is billed exactly once (ENG-161). Falls back to
	// sequential writes when the txManager/concrete repos are unavailable (e.g.
	// mock-repo tests), where there is no cross-instance concurrency.
	if s.txManager != nil && s.invRepoImpl != nil && s.subRepoImpl != nil {
		if err := s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
			won, err := s.subRepoImpl.ActivateTrialWithTx(ctx, tx, sub)
			if err != nil {
				return err
			}
			if !won {
				return errTrialRaceLost // roll back; another runner already converted
			}
			if err := s.invRepoImpl.CreateWithTx(ctx, tx, invoice); err != nil {
				return fmt.Errorf("failed to create trial conversion invoice: %w", err)
			}
			return nil
		}); err != nil {
			if errors.Is(err, errTrialRaceLost) {
				return nil, ErrTrialAlreadyConverted
			}
			return nil, err
		}
	} else {
		if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
			return nil, fmt.Errorf("failed to create trial conversion invoice: %w", err)
		}
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return nil, fmt.Errorf("failed to activate subscription after trial: %w", err)
		}
	}

	// P25 e-invoicing AFTER the invoice is durably committed (PHASE2 #3).
	s.generateEInvoiceAfterCommit(ctx, invoice, customer)

	// Dual-write to ledger (best-effort; reconciliation covers gaps).
	if s.ledger != nil {
		if err := s.ledger.RecordInvoice(ctx, invoice); err != nil {
			s.logger.Error("ledger write failed on trial conversion — will need reconciliation",
				"error", err, "invoice_id", invID, "amount", total)
		}
	}

	// Apply any account credit to the first invoice (ENG-154).
	s.applyCreditToInvoice(ctx, invoice)

	s.logger.Info("trial converted to active",
		"subscription_id", sub.ID, "invoice_id", invID, "amount", total)

	// Notify the customer that their first invoice is due (best-effort).
	if s.notificationService != nil {
		if err := s.notificationService.SendInvoiceCreated(ctx, InvoiceData{
			CustomerName:  domain.PtrToString(customer.Name),
			CustomerEmail: customer.Email,
			InvoiceNumber: invoice.InvoiceNumber,
			InvoiceID:     invoice.ID.String(),
			Amount:        formatAmount(total, price.Currency),
			DueDate:       dueDate.Format("Jan 02, 2006"),
		}); err != nil {
			s.logger.Error("failed to send trial conversion invoice notification", "error", err, "invoice_id", invID)
		}
	}

	return invoice, nil
}

// ExtendCurrentPeriod extends a subscription's current period end by the given number of days
func (s *SubscriptionService) ExtendCurrentPeriod(ctx context.Context, tenantID, subscriptionID uuid.UUID, days int) (*domain.Subscription, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	sub.CurrentPeriodEnd = sub.CurrentPeriodEnd.AddDate(0, 0, days)
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to extend subscription period: %w", err)
	}

	return sub, nil
}
