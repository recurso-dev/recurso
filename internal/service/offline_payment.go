package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// invoicePaidMarker is the narrow slice of SubscriptionService this service
// needs to settle an invoice. Depending on the interface (rather than the
// concrete *SubscriptionService) keeps the offline-payment logic unit-testable.
type invoicePaidMarker interface {
	MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}

type OfflinePaymentService struct {
	repo          port.OfflinePaymentRepository
	gateway       port.PaymentGateway
	invoiceRepo   port.InvoiceRepository
	invoiceMarker invoicePaidMarker
}

func NewOfflinePaymentService(
	repo port.OfflinePaymentRepository,
	gateway port.PaymentGateway,
	invoiceRepo port.InvoiceRepository,
	invoiceMarker invoicePaidMarker,
) *OfflinePaymentService {
	return &OfflinePaymentService{
		repo:          repo,
		gateway:       gateway,
		invoiceRepo:   invoiceRepo,
		invoiceMarker: invoiceMarker,
	}
}

type CreateVirtualAccountInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	InvoiceID  *uuid.UUID
	Amount     int64
}

func (s *OfflinePaymentService) CreateVirtualAccount(ctx context.Context, input CreateVirtualAccountInput) (*domain.VirtualAccount, error) {
	invoiceIDStr := ""
	if input.InvoiceID != nil {
		invoiceIDStr = input.InvoiceID.String()
	}

	result, err := s.gateway.CreateVirtualAccount(ctx, input.CustomerID.String(), invoiceIDStr, input.Amount, "Payment via bank transfer")
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual account: %w", err)
	}

	va := &domain.VirtualAccount{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		CustomerID:      input.CustomerID,
		InvoiceID:       input.InvoiceID,
		AccountNumber:   result.AccountNumber,
		IFSCCode:        result.IFSCCode,
		BankName:        result.BankName,
		BeneficiaryName: result.BeneficiaryName,
		RazorpayVAID:    result.VAID,
		Status:          "active",
		AmountExpected:  input.Amount,
		AmountReceived:  0,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.CreateVirtualAccount(ctx, va); err != nil {
		return nil, fmt.Errorf("failed to save virtual account: %w", err)
	}

	return va, nil
}

func (s *OfflinePaymentService) ReconcileVirtualAccount(ctx context.Context, razorpayVAID string, amount int64, paymentID string) error {
	// Atomic increment (not read-modify-write): two concurrent credits for the
	// same VA — e.g. an invoice paid in two bank transfers, which are DISTINCT
	// webhook events the inbound dedup doesn't collapse — must both be counted.
	va, err := s.repo.IncrementAmountReceived(ctx, razorpayVAID, amount)
	if err != nil {
		return fmt.Errorf("failed to record virtual-account credit: %w", err)
	}

	// Mark linked invoice as paid. Inject the VA's tenant — MarkInvoicePaid reads
	// the invoice through the tenant-scoped repository, so without this the
	// settle fails "tenant_id missing from context" and offline/bank-transfer
	// payments never settle (ENG-145).
	if va.InvoiceID != nil && va.AmountReceived >= va.AmountExpected {
		tctx := context.WithValue(ctx, domain.TenantIDKey, va.TenantID)
		if _, err := s.invoiceMarker.MarkInvoicePaid(tctx, *va.InvoiceID); err != nil {
			return fmt.Errorf("failed to mark invoice paid: %w", err)
		}
	}

	return nil
}

type RecordOfflinePaymentInput struct {
	TenantID        uuid.UUID
	CustomerID      uuid.UUID
	InvoiceID       *uuid.UUID
	PaymentType     string
	Amount          int64
	TDSAmount       int64 // tax deducted at source by the customer (India B2B); requires a linked invoice
	Currency        string
	ReferenceNumber string
	Notes           string
	RecordedBy      string
}

func (s *OfflinePaymentService) RecordOfflinePayment(ctx context.Context, input RecordOfflinePaymentInput) (*domain.OfflinePayment, error) {
	// When linked to an invoice, validate the linkage BEFORE recording anything:
	// the invoice must belong to the same customer and be in the same currency —
	// otherwise an operator could settle customer B's invoice with customer A's
	// cash, or clear an INR invoice with a JPY amount of equal numeric value.
	var inv *domain.Invoice
	tctx := context.WithValue(ctx, domain.TenantIDKey, input.TenantID)
	if input.InvoiceID != nil {
		loaded, err := s.invoiceRepo.GetByID(tctx, *input.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load invoice: %w", err)
		}
		if loaded.CustomerID != input.CustomerID {
			return nil, fmt.Errorf("offline payment customer does not match the invoice customer")
		}
		if input.Currency != "" && !strings.EqualFold(input.Currency, loaded.Currency) {
			return nil, fmt.Errorf("offline payment currency %q does not match invoice currency %q", input.Currency, loaded.Currency)
		}
		inv = loaded
	}

	// TDS deducted by the customer settles part of the invoice without cash
	// changing hands, so it is only meaningful against a linked invoice, and
	// the deduction (plus what's already deducted) can never exceed what the
	// invoice still owes.
	if input.TDSAmount < 0 {
		return nil, fmt.Errorf("tds_amount cannot be negative")
	}
	if input.TDSAmount > 0 {
		if inv == nil {
			return nil, fmt.Errorf("tds_amount requires a linked invoice")
		}
		if outstanding := inv.Total - inv.AmountPaid - inv.TDSAmount; input.TDSAmount > outstanding {
			return nil, fmt.Errorf("tds_amount %d exceeds the invoice's outstanding balance %d", input.TDSAmount, outstanding)
		}
	}

	payment := &domain.OfflinePayment{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		CustomerID:      input.CustomerID,
		InvoiceID:       input.InvoiceID,
		PaymentType:     input.PaymentType,
		Amount:          input.Amount,
		TDSAmount:       input.TDSAmount,
		Currency:        input.Currency,
		ReferenceNumber: input.ReferenceNumber,
		Notes:           input.Notes,
		RecordedBy:      input.RecordedBy,
		RecordedAt:      time.Now(),
	}

	if err := s.repo.CreateOfflinePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to record offline payment: %w", err)
	}

	// Accumulate the deduction on the invoice BEFORE settling, so the ledger
	// posting (which reads invoice.TDSAmount when the paid transition fires)
	// books DR TDS-Receivable / CR AR for the deducted portion and a cash leg
	// net of it.
	if inv != nil && input.TDSAmount > 0 {
		inv.TDSAmount += input.TDSAmount
		if err := s.invoiceRepo.Update(tctx, inv); err != nil {
			return nil, fmt.Errorf("failed to record TDS on invoice: %w", err)
		}
	}

	// Settle the linked invoice ONLY when the payment plus the customer's TDS
	// deduction (plus anything already paid/deducted) covers the total. A short
	// payment is recorded but leaves the invoice open — previously any amount
	// marked the whole invoice paid (ENG-169).
	if inv != nil {
		if input.Amount+inv.TDSAmount+inv.AmountPaid >= inv.Total {
			if _, err := s.invoiceMarker.MarkInvoicePaid(tctx, *input.InvoiceID); err != nil {
				return nil, fmt.Errorf("failed to mark invoice paid: %w", err)
			}
		}
	}

	return payment, nil
}

func (s *OfflinePaymentService) ListVirtualAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.VirtualAccount, error) {
	return s.repo.ListVirtualAccounts(ctx, tenantID)
}

func (s *OfflinePaymentService) ListOfflinePayments(ctx context.Context, tenantID uuid.UUID) ([]*domain.OfflinePayment, error) {
	return s.repo.ListOfflinePayments(ctx, tenantID)
}
