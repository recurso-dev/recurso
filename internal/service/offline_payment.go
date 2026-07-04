package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type OfflinePaymentService struct {
	repo        port.OfflinePaymentRepository
	gateway     port.PaymentGateway
	invoiceRepo port.InvoiceRepository
	subService  *SubscriptionService
}

func NewOfflinePaymentService(
	repo port.OfflinePaymentRepository,
	gateway port.PaymentGateway,
	invoiceRepo port.InvoiceRepository,
	subService *SubscriptionService,
) *OfflinePaymentService {
	return &OfflinePaymentService{
		repo:        repo,
		gateway:     gateway,
		invoiceRepo: invoiceRepo,
		subService:  subService,
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

	// Mark linked invoice as paid
	if va.InvoiceID != nil && va.AmountReceived >= va.AmountExpected {
		if err := s.subService.MarkInvoicePaid(ctx, *va.InvoiceID); err != nil {
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

	// Mark linked invoice as paid
	if input.InvoiceID != nil {
		if err := s.subService.MarkInvoicePaid(ctx, *input.InvoiceID); err != nil {
			return nil, fmt.Errorf("failed to mark invoice paid: %w", err)
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
