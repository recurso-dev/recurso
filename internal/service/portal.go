package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// PortalService handles customer portal authentication and operations
type PortalService struct {
	customerRepo  port.CustomerRepository
	invoiceRepo   port.InvoiceRepository
	magicLinkRepo port.MagicLinkRepository
	sessionRepo   port.PortalSessionRepository
	disputeRepo   port.DisputeRepository
	giftService   *GiftService
	emailSender   port.EmailSender
	portalBaseURL string
}

func NewPortalService(
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
	magicLinkRepo port.MagicLinkRepository,
	sessionRepo port.PortalSessionRepository,
	disputeRepo port.DisputeRepository,
	giftService *GiftService,
	emailSender port.EmailSender,
	portalBaseURL string,
) *PortalService {
	return &PortalService{
		customerRepo:  customerRepo,
		invoiceRepo:   invoiceRepo,
		magicLinkRepo: magicLinkRepo,
		sessionRepo:   sessionRepo,
		disputeRepo:   disputeRepo,
		giftService:   giftService,
		emailSender:   emailSender,
		portalBaseURL: portalBaseURL,
	}
}

// RequestMagicLink creates a magic link for customer login and emails it.
func (s *PortalService) RequestMagicLink(ctx context.Context, email string) (*domain.MagicLink, error) {
	// Portal login happens before any tenant is known, so this lookup is
	// intentionally cross-tenant. Most recent customer wins when the same
	// email exists in multiple tenants.
	customers, err := s.customerRepo.FindByEmailAcrossTenants(ctx, email)
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

	if s.emailSender != nil {
		loginURL := fmt.Sprintf("%s/portal/verify?token=%s", s.portalBaseURL, link.Token)
		msg := port.EmailMessage{
			To:      customer.Email,
			ToName:  domain.PtrToString(customer.Name),
			Subject: "Your login link",
			HTMLBody: fmt.Sprintf(
				`<p>Hi %s,</p><p>Click the link below to sign in to your billing portal. It expires in %s.</p><p><a href="%s">Sign in to the portal</a></p><p>If you didn't request this, you can ignore this email.</p>`,
				html.EscapeString(domain.PtrToString(customer.Name)), db.MagicLinkExpiry, loginURL,
			),
			TextBody: fmt.Sprintf("Sign in to your billing portal: %s (expires in %s)", loginURL, db.MagicLinkExpiry),
		}
		if err := s.emailSender.Send(ctx, msg); err != nil {
			// The link is created; delivery failure shouldn't 500 the request,
			// but it must be visible in logs.
			slog.Error("failed to send magic link email", "error", err, "customer_id", customer.ID)
		}
	}

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

	// Atomically consume the link. The conditional UPDATE is the real single-use
	// guard: if a concurrent verify already claimed it, marked is false and we
	// reject, so two requests can never both mint a session.
	marked, err := s.magicLinkRepo.MarkUsed(ctx, link.ID)
	if err != nil {
		return nil, err
	}
	if !marked {
		return nil, ErrMagicLinkUsed
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
	return s.invoiceRepo.GetByCustomerID(ctx, customerID)
}

// GetCustomer returns the customer profile
func (s *PortalService) GetCustomer(ctx context.Context, customerID uuid.UUID) (*domain.Customer, error) {
	return s.customerRepo.GetByIDPublic(ctx, customerID)
}

// UpdatePaymentMethod updates the vaulted payment method for the authenticated
// portal customer. The card metadata (brand/last4/expiry) comes from the
// gateway tokenization step on the client — no raw PAN ever reaches this path,
// mirroring the admin PUT /v1/customers/:id/payment-method flow. The customer
// ID is taken from the portal session and can never be supplied by the caller.
func (s *PortalService) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	return s.customerRepo.UpdatePaymentMethod(ctx, customerID, brand, last4, expMonth, expYear)
}

// RaiseDispute opens (or updates) a dispute/query on an invoice owned by the
// authenticated portal customer. Ownership is enforced against the invoice's
// customer_id; a mismatch is reported as not-found so invoice existence is not
// leaked. At most one OPEN dispute exists per invoice: re-raising updates the
// reason (or is a no-op when unchanged).
func (s *PortalService) RaiseDispute(ctx context.Context, customerID, invoiceID uuid.UUID, reason string) (*domain.InvoiceDispute, error) {
	inv, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil || inv == nil || inv.CustomerID != customerID {
		return nil, ErrInvoiceNotFound
	}

	existing, err := s.disputeRepo.GetOpenByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if reason != "" && reason != existing.Reason {
			if err := s.disputeRepo.UpdateReason(ctx, existing.ID, reason); err != nil {
				return nil, err
			}
			existing.Reason = reason
		}
		return existing, nil
	}

	dispute := &domain.InvoiceDispute{
		ID:         uuid.New(),
		TenantID:   inv.TenantID,
		InvoiceID:  invoiceID,
		CustomerID: customerID,
		Reason:     reason,
		Status:     domain.DisputeStatusOpen,
	}
	if err := s.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, err
	}
	return dispute, nil
}

// GetCustomerDisputes returns all disputes raised by the customer, so the
// portal can show dispute status alongside invoices.
func (s *PortalService) GetCustomerDisputes(ctx context.Context, customerID uuid.UUID) ([]*domain.InvoiceDispute, error) {
	return s.disputeRepo.ListByCustomerID(ctx, customerID)
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
	ErrInvoiceNotFound  = PortalError("invoice not found")
)
