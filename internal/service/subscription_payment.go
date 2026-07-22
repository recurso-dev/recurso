package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MarkInvoicePaid settles an invoice and returns whether THIS caller performed
// the paid transition. Multiple settlers (inline checkout, gateway webhook,
// retry worker, offline payment) can all call it for the same invoice, but only
// one gets transitioned=true — callers that fire once-per-settlement side
// effects (e.g. recording a dunning outcome) must gate on it so a redelivered
// webhook or a second settler doesn't double-count.
func (s *SubscriptionService) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (transitioned bool, err error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return false, err
	}
	if inv == nil {
		return false, fmt.Errorf("invoice not found")
	}

	if inv.Status == domain.InvoiceStatusPaid {
		return false, nil // Already paid
	}

	// Amount already settled by a non-cash channel before this payment — today
	// only a prepaid wallet drain at invoice generation writes amount_paid ahead
	// of settlement (partial offline payments leave the invoice open without
	// touching it). That drain already relieved AR and was booked as cash at
	// top-up, so the ledger cash leg below must EXCLUDE it; otherwise the wallet
	// portion double-books as cash and drives AR negative. Captured from the
	// freshly-loaded invoice, before MarkPaid / the AmountPaid overwrite below.
	walletSettled := inv.AmountPaid

	now := time.Now().UTC()
	// Atomically claim the paid transition. Only the settler whose conditional
	// UPDATE actually flips the row runs the side-effects below; concurrent
	// settlers get transitioned=false and return without double-posting the
	// ledger or double-counting recovered revenue.
	transitioned, err = s.invoiceRepo.MarkPaid(ctx, inv.TenantID, invoiceID, now)
	if err != nil {
		return false, err
	}
	if !transitioned {
		return false, nil // another settler already marked it paid
	}
	inv.Status = domain.InvoiceStatusPaid
	inv.PaidAt = &now
	inv.AmountPaid = inv.Total

	s.telemetry.MilestoneFirstPayment() // opt-in anonymous milestone; no-op when disabled

	// Dunning recovery attribution: if this invoice needed retries or dunning
	// to get paid, record it as recovered revenue (idempotent, non-fatal).
	if s.recoveryRecorder != nil {
		s.recoveryRecorder.RecordIfRecovered(ctx, inv)
	}

	// Record payment in ledger — cash leg net of the wallet portion already
	// settled at generation (see walletSettled above).
	if s.ledger != nil {
		if err := s.ledger.RecordPaymentWithSettled(ctx, inv, walletSettled); err != nil {
			s.logger.Error("ledger payment write failed", "error", err, "invoice_id", inv.ID)
		}
	}

	// Phase 5: Create Revenue Recognition Schedule
	if s.revrecService != nil {
		var sub *domain.Subscription
		if inv.SubscriptionID != nil {
			sub, _ = s.subRepo.GetByID(ctx, *inv.SubscriptionID)
		}
		if err := s.revrecService.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
			s.logger.Error("failed to create revrec schedule", "invoice_id", inv.ID, "error", err)
			// Don't fail the whole payment mark-paid for now, just log.
		}
	}

	// Send payment received notification
	if s.notificationService != nil {
		customer, custErr := s.customerRepo.GetByID(ctx, inv.CustomerID)
		if custErr != nil {
			s.logger.Error("failed to fetch customer for payment notification", "error", custErr, "customer_id", inv.CustomerID)
		} else if customer != nil {
			err := s.notificationService.SendPaymentReceived(ctx, PaymentData{
				CustomerName:  domain.PtrToString(customer.Name),
				CustomerEmail: customer.Email,
				InvoiceNumber: inv.InvoiceNumber,
				Amount:        formatAmount(inv.Total, inv.Currency),
				PaymentDate:   now.Format("Jan 02, 2006"),
			})
			if err != nil {
				s.logger.Error("failed to send payment received notification", "error", err, "invoice_id", inv.ID)
			}
		}
	}

	return true, nil
}
