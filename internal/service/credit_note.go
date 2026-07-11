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

// ErrCreditNoteValidation marks caller-correctable failures (bad request).
// Handlers should map errors wrapping this sentinel to HTTP 400.
var ErrCreditNoteValidation = errors.New("credit note validation failed")

// ErrRefundNotFound marks gateway refund webhook events whose refund id does
// not match any credit note (refunds issued outside recurso, or from before
// refund tracking existed). Webhook handlers should log and acknowledge (2xx)
// these — a non-2xx would make the gateway retry an event we can never consume.
var ErrRefundNotFound = errors.New("no credit note found for refund id")

// CreditNoteRepository is the persistence surface the service needs.
// *db.CreditNoteRepository satisfies it.
type CreditNoteRepository interface {
	Create(ctx context.Context, creditNote *domain.CreditNote) error
	List(ctx context.Context, tenantID uuid.UUID, filter domain.CreditNoteFilter) ([]*domain.CreditNote, error)
	UpdateRefund(ctx context.Context, id uuid.UUID, status domain.CreditNoteRefundStatus, refundID *string, message string) error
	SumActiveRefundsForInvoice(ctx context.Context, invoiceID uuid.UUID) (int64, error)
	// GetByRefundID resolves the credit note that owns a gateway refund id
	// (Stripe re_*, Razorpay rfnd_*). Returns (nil, nil) when none matches.
	GetByRefundID(ctx context.Context, refundID string) (*domain.CreditNote, error)
}

// creditNoteCustomerReader is the slice of the customer repository we use.
type creditNoteCustomerReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
}

// creditNoteInvoiceReader is the slice of the invoice repository we use.
// GetByIDPublic is used (with an explicit tenant check) because the request
// context does not carry the tenant id value the scoped GetByID requires.
type creditNoteInvoiceReader interface {
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
}

type CreditNoteService struct {
	repo         CreditNoteRepository
	customerRepo creditNoteCustomerReader
	invoiceRepo  creditNoteInvoiceReader
	gateway      port.PaymentGateway
	ledger       *LedgerService
	revrec       *RevRecService
	logger       *slog.Logger
}

// SetLedgerService wires the ledger for refund reversals (optional).
func (s *CreditNoteService) SetLedgerService(ledger *LedgerService) {
	s.ledger = ledger
}

// SetRevRecService wires rev-rec so a refund unwinds the still-deferred portion
// of the invoice's recognition schedule (ENG-147). Nil-safe.
func (s *CreditNoteService) SetRevRecService(revrec *RevRecService) {
	s.revrec = revrec
}

func NewCreditNoteService(
	repo CreditNoteRepository,
	customerRepo creditNoteCustomerReader,
	invoiceRepo creditNoteInvoiceReader,
	gateway port.PaymentGateway,
) *CreditNoteService {
	return &CreditNoteService{
		repo:         repo,
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
		gateway:      gateway,
		logger:       slog.Default().With("component", "credit_note_service"),
	}
}

func (s *CreditNoteService) Create(ctx context.Context, tenantID uuid.UUID, req domain.CreateCreditNoteRequest) (*domain.CreditNote, error) {
	// 1. Validate Customer belongs to Tenant
	customer, err := s.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid customer: %v", ErrCreditNoteValidation, err)
	}
	if customer.TenantID != tenantID {
		return nil, fmt.Errorf("%w: customer does not belong to tenant", ErrCreditNoteValidation)
	}

	cnType := domain.CreditNoteType(req.Type)
	if cnType == "" {
		cnType = domain.CreditNoteTypeAdjustment
	}
	if cnType != domain.CreditNoteTypeAdjustment && cnType != domain.CreditNoteTypeRefund {
		return nil, fmt.Errorf("%w: unknown credit note type %q", ErrCreditNoteValidation, req.Type)
	}

	ref := fmt.Sprintf("CN-%d", time.Now().Unix())
	cn := &domain.CreditNote{
		TenantID:     tenantID,
		CustomerID:   req.CustomerID,
		InvoiceID:    req.InvoiceID,
		Reference:    &ref,
		Amount:       req.Amount,
		Balance:      req.Amount, // adjustments are spendable credit
		Currency:     req.Currency,
		Status:       domain.CreditNoteStatusIssued,
		Reason:       req.Reason,
		Type:         cnType,
		RefundStatus: domain.RefundStatusNone,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if cnType == domain.CreditNoteTypeRefund {
		if err := s.createRefund(ctx, tenantID, req, cn); err != nil {
			return nil, err
		}
	} else if err := s.repo.Create(ctx, cn); err != nil {
		return nil, err
	}

	cn.Customer = customer // Populate customer for response
	return cn, nil
}

// createRefund validates the refund against the paid invoice, persists the
// credit note, and (when the invoice has a gateway payment id on record)
// issues the actual gateway refund. The credit note is always created; the
// refund outcome is tracked explicitly in refund_status so a failed or
// impossible refund is never silently presented as done.
func (s *CreditNoteService) createRefund(ctx context.Context, tenantID uuid.UUID, req domain.CreateCreditNoteRequest, cn *domain.CreditNote) error {
	if req.InvoiceID == nil {
		return fmt.Errorf("%w: refund credit note requires invoice_id", ErrCreditNoteValidation)
	}

	inv, err := s.invoiceRepo.GetByIDPublic(ctx, *req.InvoiceID)
	if err != nil {
		return fmt.Errorf("failed to load invoice %s: %w", *req.InvoiceID, err)
	}
	if inv == nil || inv.TenantID != tenantID {
		return fmt.Errorf("%w: invoice not found", ErrCreditNoteValidation)
	}
	if inv.CustomerID != req.CustomerID {
		return fmt.Errorf("%w: invoice does not belong to customer", ErrCreditNoteValidation)
	}
	if inv.Status != domain.InvoiceStatusPaid {
		return fmt.Errorf("%w: refunds can only be issued against paid invoices (invoice status is %q)", ErrCreditNoteValidation, inv.Status)
	}
	if !strings.EqualFold(inv.Currency, req.Currency) {
		return fmt.Errorf("%w: currency %s does not match invoice currency %s", ErrCreditNoteValidation, req.Currency, inv.Currency)
	}

	// Over-refund guard: this refund plus all previously issued (non-failed,
	// non-void) refunds must not exceed what the customer actually paid.
	alreadyRefunded, err := s.repo.SumActiveRefundsForInvoice(ctx, inv.ID)
	if err != nil {
		return fmt.Errorf("failed to compute refunded total for invoice %s: %w", inv.ID, err)
	}
	if req.Amount+alreadyRefunded > inv.AmountPaid {
		return fmt.Errorf("%w: refund of %d exceeds refundable amount (paid %d, already refunded %d)",
			ErrCreditNoteValidation, req.Amount, inv.AmountPaid, alreadyRefunded)
	}

	// A refund returns money to the customer; it is not a spendable balance.
	cn.Balance = 0

	// Honesty rule: without a recorded gateway payment id (offline payment,
	// or invoice paid before payment ids were tracked) no API refund is
	// possible. Create the note, but say so — never pretend a refund happened.
	if inv.GatewayPaymentID == "" {
		cn.RefundStatus = domain.RefundStatusManualRequired
		cn.RefundMessage = fmt.Sprintf(
			"invoice %s has no gateway payment id on record (offline or pre-tracking payment); no gateway refund was attempted — process this refund manually",
			inv.InvoiceNumber)
		if err := s.repo.Create(ctx, cn); err != nil {
			return err
		}
		s.logger.Warn("refund credit note requires manual processing",
			"credit_note_id", cn.ID, "invoice_id", inv.ID)
		return nil
	}

	// Persist first (pending), then call the gateway, then record the outcome.
	// This ordering guarantees a gateway refund can never happen without a
	// credit note row tracking it.
	cn.RefundStatus = domain.RefundStatusPending
	if err := s.repo.Create(ctx, cn); err != nil {
		return err
	}

	if s.gateway == nil {
		s.markRefundFailed(ctx, cn, "no payment gateway configured; refund was not sent to the gateway")
		return nil
	}

	res, err := s.gateway.Refund(ctx, inv.GatewayPaymentID, cn.Amount, inv.Currency)
	if err != nil {
		s.markRefundFailed(ctx, cn, fmt.Sprintf("gateway refund failed: %v", err))
		return nil
	}

	status := domain.RefundStatusProcessed
	if strings.EqualFold(res.Status, "pending") {
		status = domain.RefundStatusPending
	}
	refundID := res.RefundID
	message := fmt.Sprintf("gateway refund %s (gateway status: %s)", res.RefundID, res.Status)

	if err := s.repo.UpdateRefund(ctx, cn.ID, status, &refundID, message); err != nil {
		// The money moved at the gateway but we could not record it — surface
		// loudly instead of returning a note that claims otherwise.
		return fmt.Errorf("refund %s succeeded at gateway but persisting it on credit note %s failed: %w",
			res.RefundID, cn.ID, err)
	}
	cn.RefundStatus = status

	// Ledger reversal: debit Refunds, credit Cash. Same soft-fail stance as
	// invoice/payment postings — the reconciliation job surfaces any gap.
	if s.ledger != nil {
		if err := s.ledger.RecordRefund(ctx, tenantID, cn.ID, cn.Amount, "Refund for invoice "+inv.InvoiceNumber); err != nil {
			s.logger.Error("ledger refund write failed", "error", err, "credit_note_id", cn.ID)
		}
	}

	// Rev-rec unwind: reverse the still-deferred portion of this invoice's
	// recognition schedule and void the matching future events, so a mid-period
	// refund doesn't keep recognizing revenue the customer got back (ENG-147).
	// Best-effort: a failure is logged, never fails the refund.
	if s.revrec != nil {
		if reversed, err := s.revrec.UnwindOnRefund(ctx, tenantID, inv.ID, cn.ID, cn.Amount); err != nil {
			s.logger.Error("rev-rec unwind on refund failed", "error", err, "credit_note_id", cn.ID)
		} else if reversed > 0 {
			s.logger.Info("rev-rec deferred reversed on refund", "credit_note_id", cn.ID, "amount", reversed)
		}
	}
	cn.RefundID = &refundID
	cn.RefundMessage = message

	s.logger.Info("gateway refund issued for credit note",
		"credit_note_id", cn.ID,
		"invoice_id", inv.ID,
		"payment_id", inv.GatewayPaymentID,
		"refund_id", res.RefundID,
		"amount", cn.Amount,
		"status", status,
	)
	return nil
}

// markRefundFailed flags the already-created credit note as refund_failed so
// it is never mistaken for a completed refund.
func (s *CreditNoteService) markRefundFailed(ctx context.Context, cn *domain.CreditNote, message string) {
	cn.RefundStatus = domain.RefundStatusFailed
	cn.RefundMessage = message
	if err := s.repo.UpdateRefund(ctx, cn.ID, domain.RefundStatusFailed, nil, message); err != nil {
		s.logger.Error("failed to mark credit note refund as failed",
			"credit_note_id", cn.ID, "error", err, "refund_message", message)
	}
	s.logger.Error("credit note refund failed", "credit_note_id", cn.ID, "reason", message)
}

// ProcessGatewayRefundEvent consumes a gateway refund webhook (Stripe
// charge.refunded / refund.failed, Razorpay refund.processed / refund.failed)
// and advances the owning credit note's refund_status.
//
// Only "pending" moves; every other state is a logged no-op so re-delivered
// events are harmless:
//
//	pending → processed      (success event)
//	pending → refund_failed  (failure event, gateway's reason recorded)
//
// Events whose refund id matches no credit note return ErrRefundNotFound so
// the webhook handler can acknowledge them (gateways retry non-2xx responses).
func (s *CreditNoteService) ProcessGatewayRefundEvent(ctx context.Context, refundID string, succeeded bool, gatewayReason string) error {
	if refundID == "" {
		return fmt.Errorf("%w: empty refund id", ErrRefundNotFound)
	}

	cn, err := s.repo.GetByRefundID(ctx, refundID)
	if err != nil {
		return fmt.Errorf("failed to look up credit note for refund %s: %w", refundID, err)
	}
	if cn == nil {
		return fmt.Errorf("%w: %s", ErrRefundNotFound, refundID)
	}

	if cn.RefundStatus != domain.RefundStatusPending {
		// Terminal state: keep it. Re-delivered success events land here
		// (already processed), as would a late failure event after success was
		// recorded — the stored status stays authoritative.
		s.logger.Info("refund webhook ignored — credit note is not pending",
			"credit_note_id", cn.ID,
			"refund_id", refundID,
			"refund_status", cn.RefundStatus,
			"event_success", succeeded,
		)
		return nil
	}

	status := domain.RefundStatusProcessed
	message := fmt.Sprintf("gateway confirmed refund %s via webhook", refundID)
	if !succeeded {
		status = domain.RefundStatusFailed
		reason := gatewayReason
		if reason == "" {
			reason = "no reason provided by gateway"
		}
		message = fmt.Sprintf("gateway reported refund %s failed: %s", refundID, reason)
	}

	if err := s.repo.UpdateRefund(ctx, cn.ID, status, &refundID, message); err != nil {
		return fmt.Errorf("failed to persist refund %s outcome on credit note %s: %w", refundID, cn.ID, err)
	}

	s.logger.Info("credit note refund status advanced via webhook",
		"credit_note_id", cn.ID,
		"refund_id", refundID,
		"refund_status", status,
	)
	return nil
}

func (s *CreditNoteService) List(ctx context.Context, tenantID uuid.UUID, filter domain.CreditNoteFilter) ([]*domain.CreditNote, error) {
	cns, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Hydrate Customers
	// Optimization: Fetch all needed customers in one go if this becomes slow.
	// For now, simple loop is fine for MVP.
	for _, cn := range cns {
		customer, err := s.customerRepo.GetByID(ctx, cn.CustomerID)
		if err != nil {
			s.logger.Warn("credit note list: customer hydration failed",
				"credit_note_id", cn.ID, "customer_id", cn.CustomerID, "error", err)
		} else if customer != nil {
			cn.Customer = customer
		}
	}

	return cns, nil
}
