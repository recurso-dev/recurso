package service

import (
	"context"
	"fmt"
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
	va, err := s.repo.GetVirtualAccountByRazorpayID(ctx, razorpayVAID)
	if err != nil {
		return fmt.Errorf("virtual account not found: %w", err)
	}

	va.AmountReceived += amount

	if va.AmountReceived >= va.AmountExpected {
		now := time.Now()
		va.Status = "closed"
		va.ClosedAt = &now
	}

	if err := s.repo.UpdateVirtualAccount(ctx, va); err != nil {
		return fmt.Errorf("failed to update virtual account: %w", err)
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
	Currency        string
	ReferenceNumber string
	Notes           string
	RecordedBy      string
}

func (s *OfflinePaymentService) RecordOfflinePayment(ctx context.Context, input RecordOfflinePaymentInput) (*domain.OfflinePayment, error) {
	payment := &domain.OfflinePayment{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		CustomerID:      input.CustomerID,
		InvoiceID:       input.InvoiceID,
		PaymentType:     input.PaymentType,
		Amount:          input.Amount,
		Currency:        input.Currency,
		ReferenceNumber: input.ReferenceNumber,
		Notes:           input.Notes,
		RecordedBy:      input.RecordedBy,
		RecordedAt:      time.Now(),
	}

	if err := s.repo.CreateOfflinePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to record offline payment: %w", err)
	}

	// Settle the linked invoice ONLY when the payment (plus anything already
	// paid) covers the total. Previously any amount marked the whole invoice
	// paid — a typo (₹100 recorded against a ₹10,000 invoice) silently wrote off
	// the ₹9,900 balance. A short payment is now recorded but leaves the invoice
	// open. Tenant is injected because the invoice repo is tenant-scoped (ENG-145).
	if input.InvoiceID != nil {
		tctx := context.WithValue(ctx, domain.TenantIDKey, input.TenantID)
		inv, err := s.invoiceRepo.GetByID(tctx, *input.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load invoice for settlement: %w", err)
		}
		if input.Amount+inv.AmountPaid >= inv.Total {
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
