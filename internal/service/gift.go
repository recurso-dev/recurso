package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// maxGiftDurationMonths bounds a gift's term. It keeps price.Amount *
// durationMonths well clear of int64 overflow and rejects absurd inputs.
const maxGiftDurationMonths = 120

// ErrInvalidGiftDuration is returned when a gift's duration is not a positive
// number of months within the allowed range. A non-positive duration would
// produce a negative buyer invoice (crediting the buyer) and a negative-term
// gift; an unbounded one risks integer overflow on the invoice amount.
var ErrInvalidGiftDuration = errors.New("gift duration_months must be between 1 and 120")

type GiftService struct {
	giftRepo            port.GiftRepository
	subscriptionRepo    port.SubscriptionRepository
	invoiceService      *InvoiceService
	planRepo            port.PlanRepository
	notificationService *NotificationService
	creditNotes         *CreditNoteService // nil-safe; powers cancel-with-credit
}

// SetCreditNoteService wires credit issuance for gift cancellation (the buyer
// of a paid, unredeemed gift gets spendable account credit). Nil-safe: without
// it, canceling a paid gift is refused rather than silently uncompensated.
func (s *GiftService) SetCreditNoteService(cn *CreditNoteService) { s.creditNotes = cn }

func NewGiftService(
	giftRepo port.GiftRepository,
	subscriptionRepo port.SubscriptionRepository,
	invoiceService *InvoiceService,
	planRepo port.PlanRepository,
	notificationService *NotificationService,
) *GiftService {
	return &GiftService{
		giftRepo:            giftRepo,
		subscriptionRepo:    subscriptionRepo,
		invoiceService:      invoiceService,
		planRepo:            planRepo,
		notificationService: notificationService,
	}
}

// PurchaseGift creates a new Gift record, generates a buyer invoice, and notifies the recipient.
func (s *GiftService) PurchaseGift(ctx context.Context, tenantID uuid.UUID, buyerID uuid.UUID, planID uuid.UUID, recipientEmail string, durationMonths int) (*domain.Gift, error) {
	if durationMonths < 1 || durationMonths > maxGiftDurationMonths {
		return nil, ErrInvalidGiftDuration
	}

	// 1. Fetch plan to calculate price
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan has no prices")
	}

	// 2. Generate Code
	codeBytes := make([]byte, 4)
	if _, err := rand.Read(codeBytes); err != nil {
		return nil, err
	}
	code := fmt.Sprintf("GIFT-%s", hex.EncodeToString(codeBytes))

	// 3. Create Gift
	gift := &domain.Gift{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Code:            code,
		PlanID:          planID,
		BuyerCustomerID: buyerID,
		RecipientEmail:  recipientEmail,
		Status:          domain.GiftStatusPurchased,
		DurationMonths:  durationMonths,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.giftRepo.Create(ctx, gift); err != nil {
		return nil, err
	}

	// 4. Create buyer invoice for plan price * duration
	if s.invoiceService != nil {
		price := plan.Prices[0]
		giftAmount := price.Amount * int64(durationMonths)

		now := time.Now()
		invID := uuid.New()
		giftDesc := fmt.Sprintf("Gift: %s (%d month(s))", plan.Name, durationMonths)
		inv := &domain.Invoice{
			ID:            invID,
			TenantID:      tenantID,
			CustomerID:    buyerID,
			BillingReason: domain.BillingReasonGiftPurchase,
			Status:        domain.InvoiceStatusOpen,
			Currency:      price.Currency,
			Subtotal:      giftAmount,
			Total:         giftAmount,
			// Itemization (Phase 1): single line, no tax on gift purchases.
			LineItems: []domain.InvoiceItem{
				newInvoiceLine(invID, giftDesc, "", durationMonths, price.Amount, giftAmount, InvoiceTax{}, time.Time{}),
			},
			CreatedAt:    now,
			DueDate:      now,
			PaymentTerms: "net0",
		}

		if err := s.invoiceService.InvoiceRepo.Create(ctx, inv); err != nil {
			slog.Warn("failed to create gift buyer invoice", "error", err, "gift_id", gift.ID)
		} else {
			// Post the buyer invoice's double-entry leg, like every other
			// invoice-creating path. A gift purchase is a one-off (no
			// subscription), so this books DR AR / CR Revenue immediately.
			// Without it the buyer's payment posts a cash leg (CR AR) with no
			// originating debit → AR negative, gift revenue never recognized.
			s.invoiceService.recordInvoiceLeg(ctx, inv)
			if err := s.giftRepo.SetInvoiceID(ctx, gift.ID, tenantID, invID); err != nil {
				// Best-effort: an unlinked gift can still be canceled, it just
				// can't be auto-credited (the operator issues credit manually).
				slog.Warn("failed to link gift purchase invoice", "error", err, "gift_id", gift.ID)
			} else {
				gift.InvoiceID = &invID
			}
		}
	}

	// 5. Send recipient notification email
	if s.notificationService != nil && recipientEmail != "" {
		duration := fmt.Sprintf("%d month(s)", durationMonths)
		emailErr := s.notificationService.SendGiftPurchased(ctx, GiftPurchasedData{
			RecipientEmail: recipientEmail,
			PlanName:       plan.Name,
			Duration:       duration,
			GiftCode:       code,
			RedeemURL:      fmt.Sprintf("%s/portal/redeem?code=%s", s.notificationService.baseURL, code),
		})
		if emailErr != nil {
			slog.Warn("failed to send gift notification email", "error", emailErr, "gift_id", gift.ID)
		}
	}

	return gift, nil
}

// ListGifts returns gifts for a tenant with pagination
func (s *GiftService) ListGifts(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Gift, error) {
	return s.giftRepo.List(ctx, tenantID, limit, offset)
}

// RedeemGift activates the gift for a recipient
func (s *GiftService) RedeemGift(ctx context.Context, tenantID uuid.UUID, recipientCustomerID uuid.UUID, code string) (*domain.Subscription, error) {
	// 1. Find Gift
	gift, err := s.giftRepo.GetByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}
	if gift == nil {
		return nil, errors.New("invalid gift code")
	}

	if gift.Status == domain.GiftStatusRedeemed {
		return nil, errors.New("gift already redeemed")
	}

	// 2. Atomically CLAIM the gift (purchased -> redeemed) before minting the
	// subscription. Without this, two concurrent redemptions of the same code
	// both pass the status check above and each create a subscription.
	now := time.Now()
	claimed, err := s.giftRepo.MarkRedeemed(ctx, gift.ID, tenantID, recipientCustomerID, now)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errors.New("gift already redeemed")
	}
	// From here on, revert the claim on any failure so the recipient can retry.

	// 3. Create Subscription
	plan, err := s.planRepo.GetByID(ctx, gift.PlanID)
	if err != nil {
		_ = s.giftRepo.RevertRedemption(ctx, gift.ID, tenantID)
		return nil, err
	}
	if plan == nil {
		_ = s.giftRepo.RevertRedemption(ctx, gift.ID, tenantID)
		return nil, errors.New("gift plan not found")
	}

	startTime := time.Now()
	// Calculate End Time based on duration
	endTime := startTime.AddDate(0, gift.DurationMonths, 0)

	sub := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		CustomerID:         recipientCustomerID,
		PlanID:             gift.PlanID,
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: startTime,
		CurrentPeriodEnd:   endTime,
		BillingAnchor:      startTime,
		ReferenceID:        fmt.Sprintf("GIFT:%s", gift.Code), // Track origin
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		// A gift is prepaid by the BUYER for exactly gift.DurationMonths; the
		// recipient provides no payment method and never agreed to be billed.
		// CancelAtPeriodEnd makes the renewal worker END the subscription when
		// the gift period closes (renewal.go:145 → cancel, not invoice) instead
		// of generating a renewal invoice that would dun the recipient for a
		// gift. The recipient keeps full service for the gift window, then it
		// cleanly expires.
		CancelAtPeriodEnd: true,
	}

	if err := s.subscriptionRepo.Create(ctx, sub); err != nil {
		// Roll the gift back to purchased so the claim doesn't strand it.
		_ = s.giftRepo.RevertRedemption(ctx, gift.ID, tenantID)
		return nil, err
	}

	// The gift was already marked redeemed by the atomic claim above.
	return sub, nil
}

// Gift-cancellation errors; the handler maps them to HTTP statuses.
var (
	ErrGiftNotFound        = errors.New("gift not found")
	ErrGiftAlreadyRedeemed = errors.New("a redeemed gift cannot be canceled")
	ErrGiftAlreadyCanceled = errors.New("gift is already canceled")
	ErrGiftCreditUnwired   = errors.New("credit issuance is not configured; cannot cancel a paid gift")
	// ErrGiftCanceledCreditFailed marks the partial-success outcome: the gift IS
	// canceled, but compensating the buyer failed and needs a manual credit
	// note. Handlers must surface this message verbatim — hiding it behind a
	// generic 500 would leave the operator unaware a manual step is owed.
	ErrGiftCanceledCreditFailed = errors.New("gift canceled, but issuing the buyer credit failed — issue a manual credit note")
)

// GiftCancelResult reports what canceling did with the buyer's money.
type GiftCancelResult struct {
	Gift *domain.Gift `json:"gift"`
	// CreditNote is the spendable account credit issued to the buyer when the
	// purchase invoice was PAID (policy: cancel-with-credit). Nil otherwise.
	CreditNote *domain.CreditNote `json:"credit_note,omitempty"`
	// InvoiceVoided is true when the purchase invoice was still open (unpaid)
	// and was voided instead — no money had arrived, so nothing is credited.
	InvoiceVoided bool `json:"invoice_voided"`
}

// CancelGift cancels an unredeemed gift (policy decision: account credit).
// The atomic purchased→canceled transition is the single-cancel gate: only the
// winner acts on the money, so a redelivered cancel can't double-credit, and a
// cancel racing a redemption loses cleanly. What happens to the buyer's money
// follows the purchase invoice's actual state:
//   - PAID    → issue a spendable adjustment credit note for the amount paid
//     (through CreditNoteService.Create, so approval governance and
//     the GL issuance legs apply exactly as for any manual credit).
//   - open    → void the invoice; no money arrived, nothing to credit.
//   - no link → status change only (pre-link gifts); the operator issues any
//     compensation manually. Logged.
func (s *GiftService) CancelGift(ctx context.Context, tenantID, giftID, actorID uuid.UUID, actorRole string) (*GiftCancelResult, error) {
	gift, err := s.giftRepo.GetByID(ctx, tenantID, giftID)
	if err != nil {
		return nil, err
	}
	if gift == nil {
		return nil, ErrGiftNotFound
	}
	switch gift.Status {
	case domain.GiftStatusRedeemed:
		return nil, ErrGiftAlreadyRedeemed
	case domain.GiftStatusCanceled:
		return nil, ErrGiftAlreadyCanceled
	}

	// Fail-closed pre-check: a gift whose purchase is already paid must not be
	// canceled at all when credit issuance isn't wired — refuse BEFORE the
	// status flips rather than cancel-and-strand. (The money side itself is
	// decided AFTER the flip, from the invoice's then-current state.)
	if gift.InvoiceID != nil && s.invoiceService != nil {
		inv, err := s.invoiceService.InvoiceRepo.GetByIDPublic(ctx, *gift.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("load gift purchase invoice: %w", err)
		}
		if inv != nil && inv.Status == domain.InvoiceStatusPaid && s.creditNotes == nil {
			return nil, ErrGiftCreditUnwired
		}
	}

	won, err := s.giftRepo.Cancel(ctx, giftID, tenantID)
	if err != nil {
		return nil, err
	}
	if !won {
		// Lost the race: re-read to report the precise reason.
		latest, _ := s.giftRepo.GetByID(ctx, tenantID, giftID)
		if latest != nil && latest.Status == domain.GiftStatusRedeemed {
			return nil, ErrGiftAlreadyRedeemed
		}
		return nil, ErrGiftAlreadyCanceled
	}
	gift.Status = domain.GiftStatusCanceled

	result := &GiftCancelResult{Gift: gift}
	if gift.InvoiceID == nil || s.invoiceService == nil {
		slog.Warn("gift canceled without a linked purchase invoice — no automatic compensation",
			"gift_id", gift.ID, "buyer_customer_id", gift.BuyerCustomerID)
		return result, nil
	}

	// Money side, decided from the invoice's POST-cancel state — never from the
	// pre-check read. A checkout payment can settle between that read and the
	// cancel; acting on the stale read would void-fail silently and strand the
	// buyer's money. Void first (atomic on status=open), and only what is still
	// paid after the void refuses gets credited — which also means a payment
	// that was ACH-reversed back to open is voided, not credited.
	voided, err := s.invoiceService.InvoiceRepo.VoidIfOpen(ctx, tenantID, *gift.InvoiceID)
	if err != nil {
		slog.Error("gift canceled but voiding the purchase invoice failed",
			"gift_id", gift.ID, "invoice_id", *gift.InvoiceID, "error", err)
		return nil, fmt.Errorf("gift canceled, but resolving the purchase invoice failed: %w", err)
	}
	if voided {
		result.InvoiceVoided = true
		return result, nil
	}

	inv, err := s.invoiceService.InvoiceRepo.GetByIDPublic(ctx, *gift.InvoiceID)
	if err != nil {
		slog.Error("gift canceled but re-reading the purchase invoice failed — verify the buyer was compensated",
			"gift_id", gift.ID, "invoice_id", *gift.InvoiceID, "error", err)
		return nil, fmt.Errorf("gift canceled, but resolving the purchase invoice failed: %w", err)
	}
	if inv == nil || inv.Status != domain.InvoiceStatusPaid {
		// Not open (void failed), not paid: already void/canceled — nothing owed.
		return result, nil
	}

	if s.creditNotes == nil {
		// Payment landed after the fail-closed pre-check (cancel raced a
		// checkout) and credit issuance isn't wired. Never leave this silent.
		slog.Error("gift canceled but credit issuance is UNWIRED — issue the buyer's credit manually",
			"gift_id", gift.ID, "invoice_id", *gift.InvoiceID, "amount", inv.Total)
		return nil, fmt.Errorf("%w (credit issuance is not configured)", ErrGiftCanceledCreditFailed)
	}
	cn, err := s.creditNotes.Create(ctx, tenantID, actorID, actorRole, domain.CreateCreditNoteRequest{
		CustomerID: gift.BuyerCustomerID,
		InvoiceID:  gift.InvoiceID,
		Amount:     inv.Total,
		Currency:   inv.Currency,
		Reason:     fmt.Sprintf("Gift %s canceled — purchase credited", gift.Code),
		Type:       "adjustment",
	})
	if err != nil {
		// The gift is canceled but the credit failed — surface loudly; the
		// operator retries via a manual credit note. Never leave this silent.
		slog.Error("gift canceled but credit issuance FAILED — issue the buyer's credit manually",
			"gift_id", gift.ID, "invoice_id", *gift.InvoiceID, "amount", inv.Total, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrGiftCanceledCreditFailed, err)
	}
	result.CreditNote = cn
	return result, nil
}
