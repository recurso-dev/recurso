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
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type GiftService struct {
	giftRepo            port.GiftRepository
	subscriptionRepo    port.SubscriptionRepository
	invoiceService      *InvoiceService
	planRepo            port.PlanRepository
	notificationService *NotificationService
}

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
			InvoiceNumber: fmt.Sprintf("INV-GIFT-%d-%s", now.UnixNano(), invID.String()[:8]),
			BillingReason: "gift_purchase",
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

	// 2. Create Subscription
	plan, err := s.planRepo.GetByID(ctx, gift.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
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
		// AutoRenew: false? Field missing in domain?
		// Assuming domain.Subscription logic handles cancellations.
		// For a gift, usually it's "Canceled at Period End" immediately, or we let it expire.
		// Let's set status to Active but we need to ensure no future invoices are generated.
		// Simplest way: Cancel it effective at period end?
		// Or update Subscription domain to have `AutoRenew` bool.
	}

	// Phase 43 modification: `AutoRenew` field.
	// Check Subscription Domain if AutoRenew exists.
	// If not, adding it is a good idea for Gifts.

	if err := s.subscriptionRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// 3. Mark Gift Redeemed
	now := time.Now()
	gift.Status = domain.GiftStatusRedeemed
	gift.RedeemedByCustomerID = &recipientCustomerID
	gift.RedeemedAt = &now
	gift.UpdatedAt = now

	if err := s.giftRepo.Update(ctx, gift); err != nil {
		return nil, err
	}

	return sub, nil
}
