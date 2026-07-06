package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// EInvoiceService handles e-invoice generation, retry, and cancellation.
type EInvoiceService struct {
	gspAdapter    port.GSPAdapter
	invoiceRepo   port.InvoiceRepository
	customerRepo  port.CustomerRepository
	irpConfigRepo *db.IRPConfigRepository
	gstConfigRepo *db.GSTConfigRepository
	logger        *slog.Logger
}

func NewEInvoiceService(
	gspAdapter port.GSPAdapter,
	invoiceRepo port.InvoiceRepository,
	customerRepo port.CustomerRepository,
	irpConfigRepo *db.IRPConfigRepository,
	gstConfigRepo *db.GSTConfigRepository,
) *EInvoiceService {
	return &EInvoiceService{
		gspAdapter:    gspAdapter,
		invoiceRepo:   invoiceRepo,
		customerRepo:  customerRepo,
		irpConfigRepo: irpConfigRepo,
		gstConfigRepo: gstConfigRepo,
		logger:        slog.Default().With("service", "einvoice"),
	}
}

// GenerateEInvoice checks eligibility and generates an e-invoice for the given invoice.
// Returns nil response if the invoice is not eligible for e-invoicing.
func (s *EInvoiceService) GenerateEInvoice(ctx context.Context, invoice *domain.Invoice) (*port.EInvoiceResponse, error) {
	// Fetch customer
	customer, err := s.customerRepo.GetByID(ctx, invoice.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found: %s", invoice.CustomerID)
	}

	// Check eligibility
	if !s.isEligible(customer) {
		invoice.EInvoiceStatus = "NA"
		return nil, nil
	}

	// Build the e-invoice request
	req, err := s.buildRequest(ctx, invoice, customer)
	if err != nil {
		s.logger.Error("failed to build e-invoice request", "error", err, "invoice_id", invoice.ID)
		invoice.EInvoiceStatus = "FAILED"
		invoice.EInvoiceErrorMessage = fmt.Sprintf("build request failed: %v", err)
		now := time.Now().Add(5 * time.Minute)
		invoice.EInvoiceNextRetryAt = &now
		return nil, err
	}

	// Call GSP adapter
	resp, err := s.gspAdapter.GenerateIRNFull(ctx, req)
	if err != nil {
		s.logger.Error("e-invoice generation failed", "error", err, "invoice_id", invoice.ID)
		invoice.EInvoiceStatus = "FAILED"
		invoice.EInvoiceErrorMessage = err.Error()
		// Schedule first retry
		now := time.Now().Add(5 * time.Minute)
		invoice.EInvoiceNextRetryAt = &now
		if resp != nil {
			invoice.EInvoiceErrorMessage = resp.ErrorMessage
		}
		return resp, err
	}

	// Update invoice with IRN data
	invoice.IRN = resp.IRN
	invoice.AckNo = resp.AckNo
	invoice.AckDate = resp.AckDate
	invoice.SignedQRCode = resp.SignedQRCode
	invoice.EInvoiceStatus = "GENERATED"
	invoice.EInvoiceErrorMessage = ""
	invoice.EInvoiceNextRetryAt = nil

	return resp, nil
}

// RetryFailedEInvoice retries e-invoice generation for a FAILED invoice.
func (s *EInvoiceService) RetryFailedEInvoice(ctx context.Context, invoiceID uuid.UUID) (*port.EInvoiceResponse, error) {
	invoice, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, fmt.Errorf("invoice not found: %s", invoiceID)
	}

	if invoice.EInvoiceStatus != "FAILED" {
		return nil, fmt.Errorf("invoice e-invoice status is %s, expected FAILED", invoice.EInvoiceStatus)
	}

	// Reset retry tracking for manual retry
	invoice.EInvoiceRetryCount++

	resp, err := s.GenerateEInvoice(ctx, invoice)
	if err != nil {
		// Update invoice with error state
		if updateErr := s.invoiceRepo.Update(ctx, invoice); updateErr != nil {
			s.logger.Error("failed to update invoice after retry failure", "error", updateErr, "invoice_id", invoiceID)
		}
		return resp, err
	}

	// Success — persist
	if updateErr := s.invoiceRepo.Update(ctx, invoice); updateErr != nil {
		s.logger.Error("failed to update invoice after retry success", "error", updateErr, "invoice_id", invoiceID)
		return resp, updateErr
	}

	return resp, nil
}

// CancelEInvoice cancels an IRN within the 24-hour window.
func (s *EInvoiceService) CancelEInvoice(ctx context.Context, invoiceID uuid.UUID, cancelCode int, reason string) error {
	invoice, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return fmt.Errorf("invoice not found: %s", invoiceID)
	}

	if invoice.EInvoiceStatus != "GENERATED" {
		return fmt.Errorf("cannot cancel e-invoice with status %s", invoice.EInvoiceStatus)
	}

	if invoice.IRN == "" {
		return fmt.Errorf("invoice has no IRN to cancel")
	}

	// Validate cancel code (1-4)
	if cancelCode < 1 || cancelCode > 4 {
		return fmt.Errorf("invalid cancel code %d, must be 1-4", cancelCode)
	}

	// Check 24-hour window
	if invoice.AckDate != "" {
		// AckDate format from NIC: "dd/mm/yyyy hh:mm:ss" or similar
		// We do a simple time check based on invoice creation
		if time.Since(invoice.CreatedAt) > 24*time.Hour {
			return fmt.Errorf("cancellation window (24 hours) has expired")
		}
	}

	// Call GSP adapter
	if err := s.gspAdapter.CancelIRN(ctx, invoice.IRN, reason); err != nil {
		return fmt.Errorf("failed to cancel IRN: %w", err)
	}

	// Update invoice
	invoice.EInvoiceStatus = "CANCELLED"
	invoice.EInvoiceErrorMessage = ""
	invoice.EInvoiceNextRetryAt = nil

	if err := s.invoiceRepo.Update(ctx, invoice); err != nil {
		return fmt.Errorf("failed to update invoice after cancellation: %w", err)
	}

	s.logger.Info("e-invoice cancelled", "invoice_id", invoiceID, "irn", invoice.IRN, "cancel_code", cancelCode)
	return nil
}

// GetEInvoiceStatus returns the e-invoice status details for an invoice.
func (s *EInvoiceService) GetEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID) (map[string]interface{}, error) {
	invoice, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if invoice == nil {
		return nil, fmt.Errorf("invoice not found: %s", invoiceID)
	}

	return map[string]interface{}{
		"invoice_id":       invoice.ID,
		"invoice_number":   invoice.InvoiceNumber,
		"e_invoice_status": invoice.EInvoiceStatus,
		"irn":              invoice.IRN,
		"ack_no":           invoice.AckNo,
		"ack_date":         invoice.AckDate,
		"signed_qr_code":   invoice.SignedQRCode,
		"retry_count":      invoice.EInvoiceRetryCount,
		"next_retry_at":    invoice.EInvoiceNextRetryAt,
		"error_message":    invoice.EInvoiceErrorMessage,
	}, nil
}

// isEligible checks whether a customer qualifies for e-invoicing.
func (s *EInvoiceService) isEligible(customer *domain.Customer) bool {
	// Must be India, B2B (has GSTIN), and tax type business
	return customer.BillingAddress.Country == "India" &&
		domain.PtrToString(customer.GSTIN) != "" &&
		customer.TaxType == "business"
}

// buildRequest assembles an EInvoiceRequest from invoice and customer data.
func (s *EInvoiceService) buildRequest(ctx context.Context, invoice *domain.Invoice, customer *domain.Customer) (*port.EInvoiceRequest, error) {
	// Fetch tenant GST config for seller details
	var seller port.EInvoiceSeller
	if s.gstConfigRepo != nil {
		gstConfig, err := s.gstConfigRepo.GetByTenantID(ctx, invoice.TenantID)
		if err != nil {
			s.logger.Warn("failed to get GST config", "error", err, "tenant_id", invoice.TenantID)
		}
		if gstConfig != nil {
			seller = port.EInvoiceSeller{
				GSTIN:     gstConfig.GSTIN,
				LegalName: gstConfig.LegalName,
				TradeName: gstConfig.TradeName,
				Address:   gstConfig.Address,
				StateCode: gstConfig.StateCode,
			}
		}
	}

	// Build buyer from customer
	buyerStateCode := ""
	if customer.GSTIN != nil && len(*customer.GSTIN) >= 2 {
		buyerStateCode = (*customer.GSTIN)[:2]
	}
	if customer.PlaceOfSupply != nil {
		buyerStateCode = *customer.PlaceOfSupply
	}

	buyer := port.EInvoiceBuyer{
		GSTIN:     domain.PtrToString(customer.GSTIN),
		LegalName: domain.PtrToString(customer.Name),
		Address:   customer.BillingAddress.Line1,
		Location:  customer.BillingAddress.City,
		PinCode:   customer.BillingAddress.Zip,
		StateCode: buyerStateCode,
		Place:     buyerStateCode,
	}

	// Build single line item from invoice
	hsnCode := invoice.HSNCode
	if hsnCode == "" {
		hsnCode = domain.DefaultSACCode
	}

	items := []port.EInvoiceItem{
		{
			SlNo:        1,
			Description: "SaaS Subscription",
			HSNCode:     hsnCode,
			Quantity:    1,
			Unit:        "NOS",
			UnitPrice:   invoice.Subtotal,
			TotalAmount: invoice.Subtotal,
			TaxRate:     domain.DefaultGSTRate,
			IGSTAmount:  invoice.IGSTAmount,
			CGSTAmount:  invoice.CGSTAmount,
			SGSTAmount:  invoice.SGSTAmount,
		},
	}

	return &port.EInvoiceRequest{
		Invoice: invoice,
		Seller:  seller,
		Buyer:   buyer,
		Items:   items,
	}, nil
}
