package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type MandateService struct {
	mandateRepo  port.MandateRepository
	gateway      port.PaymentGateway
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
}

func NewMandateService(
	mandateRepo port.MandateRepository,
	gateway port.PaymentGateway,
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
) *MandateService {
	return &MandateService{
		mandateRepo:  mandateRepo,
		gateway:      gateway,
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
	}
}

// ErrCustomerPhoneRequired is returned when a UPI mandate is requested for a
// customer without a contact number — Razorpay rejects recurring registration
// links without one.
var ErrCustomerPhoneRequired = errors.New("customer phone number is required for a UPI mandate")

type CreateMandateInput struct {
	TenantID       uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	VPA            string
	MaxAmount      int64
	Frequency      string
}

type CreateMandateOutput struct {
	Mandate *domain.Mandate `json:"mandate"`
	AuthURL string          `json:"auth_url,omitempty"`
}

func (s *MandateService) CreateMandate(ctx context.Context, input CreateMandateInput) (*CreateMandateOutput, error) {
	customer, err := s.customerRepo.GetByID(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Razorpay requires a contact number on recurring registration links —
	// fail with a typed error so handlers can explain what's missing.
	if customer.Phone == "" {
		return nil, ErrCustomerPhoneRequired
	}

	result, err := s.gateway.CreateMandate(ctx, customer.Email, customer.Phone, input.VPA, input.MaxAmount, input.Frequency)
	if err != nil {
		return nil, fmt.Errorf("failed to create mandate with gateway: %w", err)
	}

	now := time.Now()
	mandate := &domain.Mandate{
		ID:                     uuid.New(),
		TenantID:               input.TenantID,
		CustomerID:             input.CustomerID,
		SubscriptionID:         input.SubscriptionID,
		MandateType:            "recurring",
		PaymentMethod:          "upi",
		VPA:                    input.VPA,
		RazorpayTokenID:        result.TokenID,
		RazorpaySubscriptionID: result.SubscriptionID,
		RazorpayCustomerID:     result.CustomerID,
		MaxAmount:              input.MaxAmount,
		Frequency:              input.Frequency,
		Status:                 domain.MandateStatusCreated,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := s.mandateRepo.Create(ctx, mandate); err != nil {
		return nil, fmt.Errorf("failed to save mandate: %w", err)
	}

	return &CreateMandateOutput{
		Mandate: mandate,
		AuthURL: result.AuthURL,
	}, nil
}

// HandleAuthorization activates a mandate when the gateway confirms the token.
// razorpayCustomerID may be empty; when present it is persisted so the token
// can later be revoked via Razorpay's customer-scoped token deletion API.
func (s *MandateService) HandleAuthorization(ctx context.Context, tokenID, razorpayCustomerID string) error {
	mandate, err := s.mandateRepo.GetByRazorpayTokenID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("mandate not found for token %s: %w", tokenID, err)
	}

	now := time.Now()
	mandate.Status = domain.MandateStatusActive
	mandate.AuthorizedAt = &now
	mandate.ActivatedAt = &now
	if razorpayCustomerID != "" {
		mandate.RazorpayCustomerID = razorpayCustomerID
	}

	return s.mandateRepo.Update(ctx, mandate)
}

func (s *MandateService) ExecuteDebit(ctx context.Context, mandate *domain.Mandate, amount int64, currency string) error {
	invoiceID := uuid.New()

	// The gateway's recurring-charge API needs the customer's contact details.
	customer, err := s.customerRepo.GetByID(ctx, mandate.CustomerID)
	if err != nil {
		return fmt.Errorf("mandate debit: load customer %s: %w", mandate.CustomerID, err)
	}

	result, err := s.gateway.ExecuteMandateDebit(ctx, port.MandateDebitRequest{
		TokenID:            mandate.RazorpayTokenID,
		RazorpayCustomerID: mandate.RazorpayCustomerID,
		Email:              customer.Email,
		Contact:            customer.Phone,
		Amount:             amount,
		Currency:           currency,
		InvoiceID:          invoiceID.String(),
	})
	if err != nil {
		return fmt.Errorf("mandate debit failed: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("mandate debit unsuccessful: %s", result.ErrorMsg)
	}

	// Create the invoice OPEN, not paid. The recurring debit captures
	// asynchronously, so the invoice is settled ONLY by the order.paid /
	// payment.captured webhook (which resolves it via notes.invoice_id). Booking
	// it paid here would record revenue that was never collected (ENG-141).
	now := time.Now()
	invoice := &domain.Invoice{
		ID:             invoiceID,
		TenantID:       mandate.TenantID,
		CustomerID:     mandate.CustomerID,
		SubscriptionID: mandate.SubscriptionID,
		InvoiceNumber:  fmt.Sprintf("MD-%s", invoiceID.String()[:8]),
		BillingReason:  "mandate_debit",
		AmountDue:      amount,
		AmountPaid:     0,
		Currency:       currency,
		Subtotal:       amount,
		Total:          amount,
		Status:         domain.InvoiceStatusOpen,
		// Itemization (Phase 1): single line for the mandate debit (no tax split).
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(invoiceID, "Mandate debit", "", 1, amount, amount, InvoiceTax{}, time.Time{}),
		},
		CreatedAt: now,
		DueDate:   now,
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return fmt.Errorf("failed to create invoice for mandate debit: %w", err)
	}

	// Capture the gateway payment id when the debit response carries a real one
	// (pay_*) — refunds are issued against it. The recurring-charge call returns
	// the pay_* id; the order.paid webhook also records it via SetGatewayPaymentID
	// (see WebhookHandler.HandleRazorpay), so this is a best-effort early capture.
	// Only a payment id is stored — an order id (order_*) would poison
	// gateway_payment_id and break refunds, so isGatewayPaymentID guards it.
	if isGatewayPaymentID(result.PaymentID) {
		invoice.GatewayPaymentID = result.PaymentID
		if err := s.invoiceRepo.SetGatewayPaymentID(ctx, invoice.ID, result.PaymentID); err != nil {
			// The debit succeeded and the invoice exists; failing the whole
			// debit here would re-run it next cycle and double-charge. Refunds
			// for this invoice fall back to manual_required instead.
			slog.Default().Error("failed to record gateway payment id for mandate debit",
				"invoice_id", invoice.ID, "payment_id", result.PaymentID, "error", err)
		}
	}

	// Advance mandate schedule
	mandate.LastDebitAt = &now
	mandate.PreDebitNotified = false
	nextDebit := s.calculateNextDebit(now, mandate.Frequency)
	mandate.NextDebitAt = &nextDebit

	return s.mandateRepo.Update(ctx, mandate)
}

// isGatewayPaymentID reports whether id is a refundable gateway payment
// identifier (Razorpay pay_*, Stripe pi_*/ch_*) rather than an order id —
// the Refund APIs of both gateways accept only payment identifiers.
func isGatewayPaymentID(id string) bool {
	return strings.HasPrefix(id, "pay_") || strings.HasPrefix(id, "pi_") || strings.HasPrefix(id, "ch_")
}

func (s *MandateService) Revoke(ctx context.Context, mandateID uuid.UUID) error {
	mandate, err := s.mandateRepo.GetByID(ctx, mandateID)
	if err != nil {
		return fmt.Errorf("mandate not found: %w", err)
	}

	if mandate.RazorpayTokenID != "" {
		if err := s.gateway.RevokeMandate(ctx, mandate.RazorpayCustomerID, mandate.RazorpayTokenID); err != nil {
			return fmt.Errorf("failed to revoke mandate at gateway: %w", err)
		}
	}

	now := time.Now()
	mandate.Status = domain.MandateStatusRevoked
	mandate.RevokedAt = &now

	return s.mandateRepo.Update(ctx, mandate)
}

func (s *MandateService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Mandate, error) {
	return s.mandateRepo.GetByID(ctx, id)
}

func (s *MandateService) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Mandate, error) {
	return s.mandateRepo.List(ctx, tenantID)
}

func (s *MandateService) calculateNextDebit(from time.Time, frequency string) time.Time {
	switch frequency {
	case "weekly":
		return from.AddDate(0, 0, 7)
	case "monthly":
		return from.AddDate(0, 1, 0)
	case "quarterly":
		return from.AddDate(0, 3, 0)
	case "yearly":
		return from.AddDate(1, 0, 0)
	default:
		return from.AddDate(0, 1, 0)
	}
}
