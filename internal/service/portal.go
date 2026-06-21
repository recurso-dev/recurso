package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

// PortalService handles customer portal authentication and operations
type PortalService struct {
	customerRepo  port.CustomerRepository
	invoiceRepo   port.InvoiceRepository
	magicLinkRepo port.MagicLinkRepository
	sessionRepo   port.PortalSessionRepository
	giftService   *GiftService
}

func NewPortalService(
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
	magicLinkRepo port.MagicLinkRepository,
	sessionRepo port.PortalSessionRepository,
	giftService *GiftService,
) *PortalService {
	return &PortalService{
		customerRepo:  customerRepo,
		invoiceRepo:   invoiceRepo,
		magicLinkRepo: magicLinkRepo,
		sessionRepo:   sessionRepo,
		giftService:   giftService,
	}
}

// RequestMagicLink creates a magic link for customer login
func (s *PortalService) RequestMagicLink(ctx context.Context, email string) (*domain.MagicLink, error) {
	// Find customer by email using filter
	filter := domain.CustomerFilter{
		Email: email,
	}
	customers, err := s.customerRepo.List(ctx, uuid.Nil, filter) // Search across all tenants
	if err != nil {
		return nil, err
	}

	if len(customers) == 0 {
		return nil, ErrCustomerNotFound
	}
	customer := customers[0]
	// Create magic link
	link := &domain.MagicLink{
		ID:         uuid.New(),
		CustomerID: customer.ID,
		Token:      db.GenerateSecureToken(),
		ExpiresAt:  time.Now().Add(db.MagicLinkExpiry),
	}

	if err := s.magicLinkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	// In production, send email here
	// For now, return the link for testing

	return link, nil
}

// VerifyMagicLink verifies a magic link and creates a session
func (s *PortalService) VerifyMagicLink(ctx context.Context, token string) (*domain.PortalSession, error) {
	link, err := s.magicLinkRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidMagicLink
	}

	if link.IsExpired() {
		return nil, ErrMagicLinkExpired
	}

	if link.IsUsed() {
		return nil, ErrMagicLinkUsed
	}

	// Mark link as used
	if err := s.magicLinkRepo.MarkUsed(ctx, link.ID); err != nil {
		return nil, err
	}

	// Create session
	session := &domain.PortalSession{
		ID:         uuid.New(),
		CustomerID: link.CustomerID,
		Token:      db.GenerateSecureToken(),
		ExpiresAt:  time.Now().Add(db.PortalSessionExpiry),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession validates a portal session token
func (s *PortalService) ValidateSession(ctx context.Context, token string) (*domain.PortalSession, error) {
	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidSession
	}

	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetCustomerInvoices returns invoices for a customer
func (s *PortalService) GetCustomerInvoices(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	// Get all invoices and filter by customer
	invoices, err := s.invoiceRepo.List(ctx, uuid.Nil) // All tenants
	if err != nil {
		return nil, err
	}

	var customerInvoices []*domain.Invoice
	for _, inv := range invoices {
		if inv.CustomerID == customerID {
			customerInvoices = append(customerInvoices, inv)
		}
	}

	return customerInvoices, nil
}

// GetCustomer returns the customer profile
func (s *PortalService) GetCustomer(ctx context.Context, customerID uuid.UUID) (*domain.Customer, error) {
	return s.customerRepo.GetByIDPublic(ctx, customerID)
}

// RedeemGift redeems a gift code for the customer
func (s *PortalService) RedeemGift(ctx context.Context, customerID uuid.UUID, code string) error {
	customer, err := s.customerRepo.GetByIDPublic(ctx, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}

	// Call GiftService
	_, err = s.giftService.RedeemGift(ctx, customer.TenantID, customerID, code)
	return err
}

// Errors
type PortalError string

func (e PortalError) Error() string { return string(e) }

const (
	ErrCustomerNotFound = PortalError("customer not found")
	ErrInvalidMagicLink = PortalError("invalid magic link")
	ErrMagicLinkExpired = PortalError("magic link has expired")
	ErrMagicLinkUsed    = PortalError("magic link has already been used")
	ErrInvalidSession   = PortalError("invalid session")
	ErrSessionExpired   = PortalError("session has expired")
)
