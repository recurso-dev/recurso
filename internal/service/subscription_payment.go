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

	// A written-off invoice can still get paid (a stale checkout link or a late
	// bank transfer) — MarkPaid deliberately allows uncollectible → paid.
	// Capture the pre-transition status so the write-off reversal (code 22/23)
	// can be recovered below BEFORE the cash leg posts; without it the payment
	// would settle a receivable the write-off already relieved, driving AR
	// negative, and the recognition schedule would drain an already-reversed
	// Deferred balance.
	wasWrittenOff := inv.Status == domain.InvoiceStatusUncollectible

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

	// Late recovery: if the invoice had been written off, re-establish AR /
	// Deferred / Tax Payable (codes 24/25, mirror of the write-off's 22/23)
	// BEFORE the cash leg — the payment must settle a real receivable. Failure
	// is logged loudly, not fatal: the invoice is correctly paid either way,
	// and the close pack's tie-out keeps an un-recovered write-off visible.
	if wasWrittenOff && s.ledger != nil {
		if err := s.ledger.RecordWriteOffRecovery(ctx, inv); err != nil {
			s.logger.ErrorContext(ctx, "write-off recovery posting failed — books understate AR/Deferred until reposted",
				"error", err, "invoice_id", inv.ID)
		}
	}

	// Record payment in ledger — cash leg net of the wallet portion already
	// settled at generation (see walletSettled above).
	if s.ledger != nil {
		if err := s.ledger.RecordPaymentWithSettled(ctx, inv, walletSettled); err != nil {
			s.logger.ErrorContext(ctx, "ledger payment write failed", "error", err, "invoice_id", inv.ID)
		}
	}

	// Phase 5: Create Revenue Recognition Schedule
	if s.revrecService != nil {
		var sub *domain.Subscription
		if inv.SubscriptionID != nil {
			sub, _ = s.subRepo.GetByID(ctx, *inv.SubscriptionID)
		}
		if err := s.revrecService.CreateScheduleForInvoice(ctx, inv, sub); err != nil {
			s.logger.ErrorContext(ctx, "failed to create revrec schedule", "invoice_id", inv.ID, "error", err)
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

// ReverseSettledPayment undoes a settlement the bank later clawed back — an ACH
// debit that cleared, settled the invoice, then bounced days later as a return
// (Inc 3c). It is the mirror of MarkInvoicePaid: reopen the invoice (paid →
// past_due) via a conditional UPDATE, and if THIS caller performed the
// transition, post the inverse ledger leg (DR AR / CR Cash, code 19). Returns
// whether this caller reopened the invoice; a redelivered return webhook gets
// reversed=false and skips the ledger so the reversal can't double-post.
//
// Once reopened, the invoice is past_due with the payment_attempt marked
// 'returned', so the dunning in-flight guard has cleared and the scheduler
// re-collects it on its normal cadence — no re-charge is triggered here.
func (s *SubscriptionService) ReverseSettledPayment(ctx context.Context, invoiceID uuid.UUID) (reversed bool, err error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return false, err
	}
	if inv == nil {
		return false, fmt.Errorf("invoice not found")
	}
	if inv.Status != domain.InvoiceStatusPaid {
		// Not currently settled — nothing to claw back (already reopened by an
		// earlier delivery of the same return, or never reached paid).
		return false, nil
	}

	// The bank returned the CASH payment; the wallet/credit/TDS portion was not
	// clawed back. Reconstruct that non-cash portion from the actual cash leg
	// (non-cash = Total − CreditApplied − TDSAmount − cashLeg) and retain it as
	// paid on reopen — otherwise re-collect posts the cash leg on the full Total
	// and drives AR negative (and dunning over-collects the wallet portion again).
	var retainPaid int64
	if s.ledger != nil {
		cashAmt, cerr := s.ledger.LatestSettledCashAmount(ctx, invoiceID)
		if cerr != nil {
			// Fail CLOSED: reopening with retainPaid=0 would discard the non-cash
			// portion and make re-collect post the cash leg on the full Total — the
			// pre-#349 double-charge + negative-AR bug. Leave the invoice paid and
			// return the error; the webhook is retried (non-2xx) until the ledger
			// read succeeds, so the reversal is retried, never corrupted.
			return false, fmt.Errorf("reversal: cash-leg lookup failed for invoice %s: %w", invoiceID, cerr)
		}
		retainPaid = inv.Total - inv.CreditApplied - inv.TDSAmount - cashAmt
		if retainPaid < 0 {
			retainPaid = 0
		}
	}

	reversed, err = s.invoiceRepo.ReverseToUnpaid(ctx, inv.TenantID, invoiceID, retainPaid)
	if err != nil {
		return false, err
	}
	if !reversed {
		return false, nil // another handler already reopened it
	}
	inv.Status = domain.InvoiceStatusPastDue
	inv.PaidAt = nil
	inv.AmountPaid = retainPaid

	// Reverse the settlement cash leg. Non-fatal per ADR-002: a failed posting is
	// caught by reconciliation, and the invoice is already correctly reopened.
	if s.ledger != nil {
		if err := s.ledger.RecordPaymentReversal(ctx, inv); err != nil {
			s.logger.ErrorContext(ctx, "ledger payment reversal write failed", "error", err, "invoice_id", inv.ID)
		}
	}

	s.logger.Warn("payment reversed (bank return); invoice reopened for collection",
		"invoice_id", inv.ID, "invoice_number", inv.InvoiceNumber)
	return true, nil
}
