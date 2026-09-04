package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ProrationResult contains the calculation for an upgrade/downgrade
type ProrationResult struct {
	CreditAmount       int64     // Amount credited for unused time on old plan
	ChargeAmount       int64     // Amount charged for remaining time on new plan
	NetAmount          int64     // Net amount to invoice (Charge - Credit)
	ProrationDate      time.Time // Date the proration occurred
	UnusedSeconds      float64   // Number of seconds unused in the period
	RemainingSeconds   float64   // Number of seconds remaining in the period
	PeriodTotalSeconds float64   // Total seconds in the period
}

// CalculateProration calculates the credit and charge amounts for a plan change
func (s *SubscriptionService) CalculateProration(
	currentPlanPrice int64,
	newPlanPrice int64,
	periodStart time.Time,
	periodEnd time.Time,
	prorationDate time.Time,
) *ProrationResult {
	totalDuration := periodEnd.Sub(periodStart).Seconds()
	if totalDuration <= 0 {
		return &ProrationResult{}
	}

	remainingDuration := periodEnd.Sub(prorationDate).Seconds()
	if remainingDuration < 0 {
		remainingDuration = 0
	}

	unusedDuration := remainingDuration // In simple terms, unused time on old plan matches remaining time on new plan

	// Calculate Credit for Old Plan (Unused Time)
	creditAmount := int64(math.Round(float64(currentPlanPrice) * (unusedDuration / totalDuration)))

	// Calculate Charge for New Plan (Remaining Time)
	chargeAmount := int64(math.Round(float64(newPlanPrice) * (remainingDuration / totalDuration)))

	return &ProrationResult{
		CreditAmount:       creditAmount,
		ChargeAmount:       chargeAmount,
		NetAmount:          chargeAmount - creditAmount,
		ProrationDate:      prorationDate,
		UnusedSeconds:      unusedDuration,
		RemainingSeconds:   remainingDuration,
		PeriodTotalSeconds: totalDuration,
	}
}

// PlanChangeProration bundles a proration result with the tax computed on its
// net amount. Both UpdateSubscription (apply) and PreviewPlanChange (preview)
// obtain it from computePlanChangeProration, guaranteeing the previewed numbers
// equal what apply will actually charge.
type PlanChangeProration struct {
	Proration     *ProrationResult
	Tax           InvoiceTax
	Currency      string
	EffectiveDate time.Time
}

// computePlanChangeProration is the single source of truth for plan-change
// math. Each side is taxed on its own plan's HSN — the remaining new-plan charge
// at the new plan's rate, the unused old-plan credit at the old plan's rate
// (ENG-150) — and the two are netted, so mixed-rate changes are taxed correctly
// and the credit refunds the GST originally collected.
func (s *SubscriptionService) computePlanChangeProration(
	ctx context.Context,
	tenantID uuid.UUID,
	sub *domain.Subscription,
	currentPlan, newPlan *domain.Plan,
	customer *domain.Customer,
	now time.Time,
) PlanChangeProration {
	if currentPlan == nil || newPlan == nil || len(currentPlan.Prices) == 0 || len(newPlan.Prices) == 0 {
		return PlanChangeProration{Proration: &ProrationResult{ProrationDate: now}, EffectiveDate: now}
	}

	currency := newPlan.Prices[0].Currency

	// R3 (ENG-195): prorate at the prices the customer actually pays THIS
	// period. A coupon-blind proration credited unused time at LIST price, so a
	// heavily-discounted subscription could downgrade into more account credit
	// than it ever paid (money-out over-credit), and an upgrading discounted
	// customer was charged the full list-price difference while renewals honor
	// the coupon. When the current period's invoice carried the discount
	// (CouponAppliedCurrentPeriod, recorded at every generation site), both
	// sides are discounted: the old-plan credit refunds what was actually paid
	// for the unused time, and the new-plan charge matches what a renewal on the
	// new plan would bill inside this discounted period.
	currentPrice := currentPlan.Prices[0].Amount
	newPrice := newPlan.Prices[0].Amount
	if sub.CouponAppliedCurrentPeriod && sub.CouponID != nil && s.couponRepo != nil {
		if coupon, cerr := s.couponRepo.GetByID(ctx, tenantID, *sub.CouponID); cerr != nil {
			s.logger.Error("plan-change proration: coupon load failed; prorating at list prices",
				"subscription_id", sub.ID, "coupon_id", *sub.CouponID, "error", cerr)
		} else {
			currentPrice -= couponDiscountFor(coupon, currentPrice)
			newPrice -= couponDiscountFor(coupon, newPrice)
		}
	}

	proration := s.CalculateProration(
		currentPrice,
		newPrice,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		now,
	)

	var taxRes InvoiceTax
	if customer != nil && proration.NetAmount != 0 {
		// Tax each side on its OWN plan's HSN, then net. The remaining-new-plan
		// charge collects GST at the new plan's rate; the unused-old-plan credit
		// reverses GST at the old plan's rate (ENG-150). Applying a single rate
		// to the net (charge − credit) — the previous behaviour — is only correct
		// when both plans resolve to the same rate; when they differ (e.g. an 18%
		// SaaS plan ↔ a 12%/5% service plan) it reversed or collected GST at the
		// wrong rate, over/under-charging the customer (ENG-158).
		chargeTax := s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, proration.ChargeAmount, newPlan.HSNCode)
		creditTax := s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, proration.CreditAmount, currentPlan.HSNCode)
		taxRes = InvoiceTax{
			Total: chargeTax.Total - creditTax.Total,
			IGST:  chargeTax.IGST - creditTax.IGST,
			CGST:  chargeTax.CGST - creditTax.CGST,
			SGST:  chargeTax.SGST - creditTax.SGST,
		}
		// Descriptive fields (type/rate/HSN) follow the side that dominates the
		// net: the new plan on a net charge (upgrade), the old plan on a net
		// credit (downgrade). The money amounts above are always the true net.
		if proration.NetAmount > 0 {
			taxRes.TaxType, taxRes.Rate, taxRes.HSN, taxRes.Note = chargeTax.TaxType, chargeTax.Rate, chargeTax.HSN, chargeTax.Note
		} else {
			taxRes.TaxType, taxRes.Rate, taxRes.HSN, taxRes.Note = creditTax.TaxType, creditTax.Rate, creditTax.HSN, creditTax.Note
		}
	}

	return PlanChangeProration{Proration: proration, Tax: taxRes, Currency: currency, EffectiveDate: now}
}

// PlanChangePreview is the read-only breakdown returned by PreviewPlanChange.
// All monetary fields are in the currency's smallest unit (e.g. paise/cents).
type PlanChangePreview struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	CurrentPlanID     uuid.UUID `json:"current_plan_id"`
	NewPlanID         uuid.UUID `json:"new_plan_id"`
	Currency          string    `json:"currency"`
	CreditAmount      int64     `json:"credit_amount"`       // credit for unused time on the current plan
	ChargeAmount      int64     `json:"charge_amount"`       // prorated charge for the remaining period on the new plan
	NetAmount         int64     `json:"net_amount"`          // charge - credit, before tax
	TaxAmount         int64     `json:"tax_amount"`          // tax on the net: positive on a charge, negative (reversed GST) on a downgrade credit (ENG-150)
	TotalAmount       int64     `json:"total_amount"`        // net + tax: the immediate proration charge (positive) or credit (negative)
	EffectiveDate     time.Time `json:"effective_date"`      // when the change would take effect (now)
	NextInvoiceAmount int64     `json:"next_invoice_amount"` // full new-plan charge incl. tax at the next renewal
	IsUpgrade         bool      `json:"is_upgrade"`          // true when the new plan costs more than the current one
}

// PreviewPlanChange computes the proration for switching a subscription to
// newPlanID WITHOUT applying it. It reuses computePlanChangeProration — the
// exact function UpdateSubscription uses — so the preview matches the charge.
func (s *SubscriptionService) PreviewPlanChange(ctx context.Context, tenantID, subscriptionID, newPlanID uuid.UUID) (*PlanChangePreview, error) {
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	if sub.TenantID != tenantID {
		return nil, ErrSubscriptionNotFound
	}

	currentPlan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	if currentPlan == nil || len(currentPlan.Prices) == 0 {
		return nil, fmt.Errorf("current plan unavailable for preview")
	}

	newPlan, err := s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, ErrPlanNotFound
	}
	if len(newPlan.Prices) == 0 {
		return nil, fmt.Errorf("new plan has no prices")
	}

	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	now := time.Now().UTC()
	pcp := s.computePlanChangeProration(ctx, tenantID, sub, currentPlan, newPlan, customer, now)

	// Resulting next-invoice amount: the full new-plan price plus tax, i.e. what
	// the customer pays at the next renewal once fully on the new plan.
	newPrice := newPlan.Prices[0].Amount
	var nextTax InvoiceTax
	if newPrice > 0 && customer != nil {
		nextTax = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, pcp.Currency, newPrice, newPlan.HSNCode)
	}

	return &PlanChangePreview{
		SubscriptionID:    sub.ID,
		CurrentPlanID:     sub.PlanID,
		NewPlanID:         newPlanID,
		Currency:          pcp.Currency,
		CreditAmount:      pcp.Proration.CreditAmount,
		ChargeAmount:      pcp.Proration.ChargeAmount,
		NetAmount:         pcp.Proration.NetAmount,
		TaxAmount:         pcp.Tax.Total,
		TotalAmount:       pcp.Proration.NetAmount + pcp.Tax.Total,
		EffectiveDate:     pcp.EffectiveDate,
		NextInvoiceAmount: newPrice + nextTax.Total,
		IsUpgrade:         newPrice > currentPlan.Prices[0].Amount,
	}, nil
}

// UpdateSubscription updates a subscription's plan and handles proration
// persistPlanChange writes the proration invoice and/or downgrade credit note
// and flips the subscription's plan. See the atomicity note at the transaction
// branch below (PHASE2 #1).
func (s *SubscriptionService) persistPlanChange(ctx context.Context, chargeInvoice *domain.Invoice, creditNote *domain.CreditNote, sub *domain.Subscription) error {
	// Atomic path: the proration invoice OR downgrade credit note commits together
	// with the plan flip in one transaction. A failed flip can never leave an
	// orphaned charge — or (the exploit) a spendable credit without the actual
	// downgrade, which a caller could loop for unbounded credit (PHASE2 #1).
	canTx := s.txManager != nil && s.subRepoImpl != nil && s.invRepoImpl != nil &&
		(creditNote == nil || s.creditNoteRepo != nil)
	if canTx {
		return s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
			if chargeInvoice != nil {
				if err := s.invRepoImpl.CreateWithTx(ctx, tx, chargeInvoice); err != nil {
					return fmt.Errorf("failed to create proration invoice: %w", err)
				}
			}
			if creditNote != nil {
				if err := s.creditNoteRepo.CreateWithTx(ctx, tx, creditNote); err != nil {
					return fmt.Errorf("failed to create downgrade credit note: %w", err)
				}
			}
			return s.subRepoImpl.UpdateWithTx(ctx, tx, sub)
		})
	}

	// Fallback for mock/partial wiring (tests without concrete repos): sequential
	// best-effort. Not atomic, but only reached when the tx path is unavailable.
	// Tightened: Update the subscription FIRST. If this fails, no credit is
	// issued. If it succeeds but the credit fails, it's bad UX but financially
	// safe (prevents the downgrade exploit).
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	if chargeInvoice != nil {
		if err := s.invoiceRepo.Create(ctx, chargeInvoice); err != nil {
			s.logger.Error("failed to create proration invoice after plan flip", "subscription_id", sub.ID, "error", err)
			return fmt.Errorf("plan updated but failed to create proration invoice: %w", err)
		}
	}
	if creditNote != nil {
		if s.creditNoteRepo != nil {
			if err := s.creditNoteRepo.Create(ctx, creditNote); err != nil {
				s.logger.Error("failed to create downgrade credit note after plan flip", "subscription_id", sub.ID, "error", err)
				return fmt.Errorf("plan updated but failed to create downgrade credit note: %w", err)
			}
		} else {
			s.logger.Warn("downgrade proration credit not persisted (no credit-note repo configured)",
				"subscription_id", sub.ID, "amount", creditNote.Amount)
		}
	}
	return nil
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, tenantID, subscriptionID, newPlanID uuid.UUID) (*domain.Subscription, error) {
	// 1. Fetch Subscription & Current Plan
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	if sub.TenantID != tenantID {
		return nil, ErrSubscriptionNotFound
	}

	// 1.5 Fetch Customer
	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	if sub.PlanID == newPlanID {
		return sub, nil // No change
	}

	currentPlan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch New Plan
	newPlan, err := s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, ErrPlanNotFound
	}

	// 3. Calculate Proration via the shared helper so apply and preview
	// (PreviewPlanChange) always agree on credit/charge/tax.
	now := time.Now().UTC()
	pcp := s.computePlanChangeProration(ctx, tenantID, sub, currentPlan, newPlan, customer, now)
	proration := pcp.Proration
	taxRes := pcp.Tax

	// 4. Build the proration record and apply the plan change atomically.
	//   - NetAmount > 0 (upgrade): issue a CHARGE invoice (tax-inclusive) and
	//     flip the plan in the same DB transaction, so a plan change can never
	//     land without its charge (or vice versa).
	//   - NetAmount < 0 (downgrade): persist the credit as a spendable
	//     adjustment CREDIT NOTE including the reversed tax. Previously the
	//     credit was force-zeroed onto a $0 "paid" invoice and silently vanished
	//     (ENG-150). The credit note is created first, then the plan flips.
	sub.PlanID = newPlanID
	sub.UpdatedAt = now

	var chargeInvoice *domain.Invoice
	var creditNote *domain.CreditNote

	switch {
	case proration.NetAmount > 0:
		prInvID := uuid.New()
		prDesc := "Plan change proration"
		if newPlan.Name != "" {
			prDesc = fmt.Sprintf("Proration: %s", newPlan.Name)
		}
		chargeInvoice = &domain.Invoice{
			ID:             prInvID,
			TenantID:       tenantID,
			EntityID:       sub.EntityID, // Multi-Entity Books: post the proration to the sub's own ledger, not the primary
			SubscriptionID: &sub.ID,
			CustomerID:     sub.CustomerID,
			BillingReason:  domain.BillingReasonSubscriptionUpdate,
			Status:         domain.InvoiceStatusOpen,
			Currency:       pcp.Currency,
			Subtotal:       proration.NetAmount,
			TaxAmount:      taxRes.Total,
			TaxType:        taxRes.TaxType, // D3c: persist for the liability report
			IGSTAmount:     taxRes.IGST,
			CGSTAmount:     taxRes.CGST,
			SGSTAmount:     taxRes.SGST,
			Total:          proration.NetAmount + taxRes.Total,
			LineItems: []domain.InvoiceItem{
				newInvoiceLine(prInvID, prDesc, taxRes.HSN, 1, proration.NetAmount, proration.NetAmount, taxRes, time.Time{}),
			},
			CreatedAt: now,
			DueDate:   now,
		}

		// P25 e-invoicing is deferred to AFTER the invoice is committed (below the
		// persist call) — see generateEInvoiceAfterCommit (PHASE2 #3).

	case proration.NetAmount < 0:
		// Both proration.NetAmount and taxRes.Total are negative here; negating
		// their sum yields a positive, spendable credit balance.
		creditAmount := -(proration.NetAmount + taxRes.Total)
		// B2 (ENG-196): record the tax breakdown so the credit-note document is a
		// statutory-grade CDN. The net values are negative on a downgrade — the
		// negations below are the tax actually reversed to the customer, clamped
		// at 0 so a mixed-rate change can never record a negative component.
		clampPos := func(v int64) int64 {
			if v < 0 {
				return 0
			}
			return v
		}
		creditNote = &domain.CreditNote{
			ID:           uuid.New(),
			TenantID:     tenantID,
			EntityID:     sub.EntityID,
			CustomerID:   sub.CustomerID,
			Amount:       creditAmount,
			Balance:      creditAmount,
			Subtotal:     clampPos(-proration.NetAmount),
			TaxAmount:    clampPos(-taxRes.Total),
			IGSTAmount:   clampPos(-taxRes.IGST),
			CGSTAmount:   clampPos(-taxRes.CGST),
			SGSTAmount:   clampPos(-taxRes.SGST),
			TaxType:      taxRes.TaxType,
			HSNCode:      taxRes.HSN,
			Currency:     pcp.Currency,
			Status:       domain.CreditNoteStatusIssued,
			Reason:       "Plan downgrade proration credit",
			Type:         domain.CreditNoteTypeAdjustment,
			RefundStatus: domain.RefundStatusNone,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	// 5. Persist. Charge path: invoice + plan flip in one transaction. Credit
	// path: credit note first (an orphaned credit is recoverable), then plan
	// flip. If the txManager/concrete repos are unavailable (e.g. tests with a
	// mock repo), fall back to sequential writes.
	if err := s.persistPlanChange(ctx, chargeInvoice, creditNote, sub); err != nil {
		return nil, err
	}

	// P25 e-invoicing runs AFTER the invoice is durably committed: registering a
	// government IRN before commit would orphan an irreversible IRN at NIC if the
	// transaction rolled back (PHASE2 #3).
	if chargeInvoice != nil {
		s.generateEInvoiceAfterCommit(ctx, chargeInvoice, customer)
	}

	// Post the upgrade charge's invoice leg (DR AR / CR Deferred/Revenue) to the
	// ledger, symmetric to the downgrade-credit posting below and to the initial
	// invoice in CreateSubscription. Without it the charge invoice would only ever
	// get its cash leg on payment, leaving AR/Deferred permanently imbalanced —
	// the reconciler flags it as missing_invoice_transaction (F1). Best-effort,
	// after commit: a post failure is logged for reconciliation, never fails the
	// plan change.
	if chargeInvoice != nil && s.ledger != nil {
		if err := s.ledger.RecordInvoice(ctx, chargeInvoice); err != nil {
			s.logger.Error("upgrade proration ledger invoice post failed — reconciliation needed",
				"invoice_id", chargeInvoice.ID, "amount", chargeInvoice.Total, "error", err)
		}
	}

	// Apply account credit to the upgrade charge invoice (ENG-154).
	if chargeInvoice != nil {
		s.applyCreditToInvoice(ctx, chargeInvoice)
	}

	// Rev-rec + ledger for a downgrade credit (ENG-154): the over-deferred
	// portion of the current period stops being revenue we'll earn and becomes
	// account credit we owe. Shrink the recognition schedule by the credit amount
	// (so it recognizes only the new plan's remaining service) and post
	// DR Deferred / CR Customer-Credit for the same amount, keeping Deferred and
	// the schedule in step. Best-effort: failures are logged for reconciliation.
	if creditNote != nil && creditNote.Amount > 0 {
		// The credit note is GROSS (net + reversed GST). Split it: the NET reduces
		// Deferred and the recognition schedule (both of which hold net-of-tax
		// after ENG-191), while the tax portion reverses out of Tax Payable — the
		// two together credit the customer the gross they paid. Passing the gross
		// to the net-holding Deferred/schedule would drive Deferred negative by
		// the tax (ENG-191c).
		netCredit := -proration.NetAmount
		if netCredit < 0 {
			netCredit = 0
		}
		taxCredit := -taxRes.Total
		if taxCredit < 0 {
			taxCredit = 0
		}
		// Split the net credit by where its funding actually sits (ENG-191d):
		//   1. the schedule's still-pending part -> DR Deferred (schedule reduced
		//      in step, so Deferred and the schedule move together);
		//   2. revenue this subscription GENUINELY recognized -> DR Recognized
		//      Revenue, capped at (and marking) its recognized events, so repeated
		//      downgrades can never claw back more than was ever recognized;
		//   3. any residual is deferred-but-unscheduled value (an unpaid
		//      upgrade-charge invoice funds Deferred but has no schedule until it
		//      is paid) -> DR Deferred, where that funding still sits.
		// Attributing the whole shortfall to Recognized Revenue drove IT
		// wrong-sign whenever the shortfall was really unscheduled deferral; the
		// old full-Deferred behavior drove Deferred wrong-sign whenever revenue
		// had genuinely recognized ahead. If revrec is unavailable we can't know
		// the split, so we keep the conservative full-Deferred behavior.
		deferredPortion := netCredit
		revenueReversal := int64(0)
		if s.revrecService != nil && netCredit > 0 {
			if reduced, err := s.revrecService.ReduceScheduleForDowngrade(ctx, tenantID, subscriptionID, netCredit); err != nil {
				s.logger.Error("downgrade schedule reduction failed", "subscription_id", subscriptionID, "error", err)
			} else if shortfall := netCredit - reduced; shortfall > 0 {
				reversed, rErr := s.revrecService.ReverseRecognizedForDowngrade(ctx, tenantID, subscriptionID, shortfall)
				if rErr != nil {
					s.logger.Error("downgrade recognized-revenue reversal failed — funding the shortfall from Deferred",
						"subscription_id", subscriptionID, "error", rErr)
				}
				revenueReversal = reversed
				deferredPortion = netCredit - reversed
				// The residual beyond schedule + recognized is UNSCHEDULED deferral
				// (an unpaid invoice's code-1 funding). Its Deferred is being debited
				// here, so when that invoice is later paid its new schedule must
				// shrink by the same amount (ENG-191f) — otherwise it would recognize
				// revenue the business already credited back and over-drain Deferred.
				if residual := shortfall - reversed; residual > 0 {
					if dErr := s.revrecService.RecordScheduleDebt(ctx, subscriptionID, residual); dErr != nil {
						s.logger.Error("downgrade schedule-debt record failed — a later schedule may over-recognize",
							"subscription_id", subscriptionID, "amount", residual, "error", dErr)
					}
				}
			} else {
				deferredPortion = netCredit
			}
		}
		if s.ledger != nil {
			if deferredPortion > 0 {
				if _, err := s.ledger.RecordDowngradeCredit(ctx, tenantID, creditNote.EntityID, creditNote.ID, deferredPortion, "Plan downgrade credit (net, deferred portion)"); err != nil {
					s.logger.Error("downgrade credit ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", deferredPortion, "error", err)
				}
			}
			if revenueReversal > 0 {
				if _, err := s.ledger.RecordDowngradeRevenueReversal(ctx, tenantID, creditNote.EntityID, creditNote.ID, revenueReversal, "Plan downgrade credit (net, recognized portion)"); err != nil {
					s.logger.Error("downgrade revenue reversal ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", revenueReversal, "error", err)
				}
			}
			if taxCredit > 0 {
				if _, err := s.ledger.RecordDowngradeTaxReversal(ctx, tenantID, creditNote.EntityID, creditNote.ID, taxCredit, "Plan downgrade GST reversal"); err != nil {
					s.logger.Error("downgrade tax reversal ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", taxCredit, "error", err)
				}
			}
		}
	}

	// Sync with Gateway (Razorpay/Stripe) — not implemented yet.
	// When s.gateway != nil && sub.RazorpaySubscriptionID != "":
	// s.gateway.UpdateSubscription(ctx, sub.RazorpaySubscriptionID, newPlan.Code)

	return sub, nil
}
