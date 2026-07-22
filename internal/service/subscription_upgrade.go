package service

import (
	"context"
	"database/sql"
	"fmt"
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
	creditAmount := int64(float64(currentPlanPrice) * (unusedDuration / totalDuration))

	// Calculate Charge for New Plan (Remaining Time)
	chargeAmount := int64(float64(newPlanPrice) * (remainingDuration / totalDuration))

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
// math. A positive net (a charge) is taxed on the new plan's HSN; a negative net
// (a downgrade credit) carries the reversed tax computed on the old plan's HSN
// (ENG-150), so the credit refunds the GST originally collected.
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
	proration := s.CalculateProration(
		currentPlan.Prices[0].Amount,
		newPlan.Prices[0].Amount,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		now,
	)

	var taxRes InvoiceTax
	if customer != nil && proration.NetAmount != 0 {
		if proration.NetAmount > 0 {
			taxRes = s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, proration.NetAmount, newPlan.HSNCode)
		} else {
			// Downgrade credit: reverse the GST collected on the old plan's unused
			// portion. Resolve on the positive base with the OLD plan's HSN, then
			// negate so the credit carries the tax reversal (ENG-150) — previously
			// a credit reversed no tax and under-refunded the customer.
			t := s.taxResolver.ResolveInvoiceTax(ctx, tenantID, customer, currency, -proration.NetAmount, currentPlan.HSNCode)
			taxRes = InvoiceTax{
				Total: -t.Total, IGST: -t.IGST, CGST: -t.CGST, SGST: -t.SGST,
				TaxType: t.TaxType, Note: t.Note, Rate: t.Rate, HSN: t.HSN,
			}
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
		return nil, fmt.Errorf("subscription not found")
	}
	if sub.TenantID != tenantID {
		return nil, fmt.Errorf("subscription not found for tenant")
	}

	// 1.5 Fetch Customer
	customer, err := s.customerRepo.GetByID(ctx, sub.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
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
		return nil, fmt.Errorf("new plan not found")
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
		creditNote = &domain.CreditNote{
			ID:           uuid.New(),
			TenantID:     tenantID,
			EntityID:     sub.EntityID,
			CustomerID:   sub.CustomerID,
			Amount:       creditAmount,
			Balance:      creditAmount,
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
		if s.revrecService != nil && netCredit > 0 {
			if reduced, err := s.revrecService.ReduceScheduleForDowngrade(ctx, tenantID, subscriptionID, netCredit); err != nil {
				s.logger.Error("downgrade schedule reduction failed", "subscription_id", subscriptionID, "error", err)
			} else if reduced != netCredit {
				s.logger.Warn("downgrade schedule reduced less than the net credit (deferred/credit may need reconciliation)",
					"subscription_id", subscriptionID, "net_credit", netCredit, "reduced", reduced)
			}
		}
		if s.ledger != nil {
			if netCredit > 0 {
				if _, err := s.ledger.RecordDowngradeCredit(ctx, tenantID, creditNote.EntityID, creditNote.ID, netCredit, "Plan downgrade credit (net)"); err != nil {
					s.logger.Error("downgrade credit ledger post failed — reconciliation needed",
						"credit_note_id", creditNote.ID, "amount", netCredit, "error", err)
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
