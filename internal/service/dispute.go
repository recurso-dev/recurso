package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// disputeInvoiceReader is the slice of the invoice repository the dispute
// service needs to size a resolution credit. GetByIDPublic is used (with an
// explicit tenant check) because the request context does not carry the tenant
// id value the scoped GetByID requires — same idiom as the credit-note service.
type disputeInvoiceReader interface {
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
}

// DisputeService provides the admin-facing operations over invoice disputes.
// The portal-facing (customer) operations live on PortalService, which owns
// the invoice-ownership guard.
type DisputeService struct {
	repo        port.DisputeRepository
	creditNotes *CreditNoteService
	invoices    disputeInvoiceReader
}

func NewDisputeService(repo port.DisputeRepository) *DisputeService {
	return &DisputeService{repo: repo}
}

// SetCreditIssuer wires the credit-note service + invoice reader so accepting a
// dispute can optionally issue a resolution credit. Nil-safe: without it, an
// accept can still close the dispute but can't issue credit.
func (s *DisputeService) SetCreditIssuer(cn *CreditNoteService, invoices disputeInvoiceReader) {
	s.creditNotes = cn
	s.invoices = invoices
}

// List returns tenant-scoped disputes, optionally filtered by status
// ("open", "resolved", "rejected"); an empty status returns all.
func (s *DisputeService) List(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]*domain.InvoiceDispute, error) {
	return s.repo.ListByTenant(ctx, tenantID, status, limit, offset)
}

// Get returns one tenant-scoped dispute, or (nil, nil) when it doesn't exist or
// belongs to another tenant — so a bad/cross-tenant id is a 404, never a leak.
func (s *DisputeService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.InvoiceDispute, error) {
	dispute, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load dispute: %w", err)
	}
	if dispute == nil || dispute.TenantID != tenantID {
		return nil, nil
	}
	return dispute, nil
}

// Resolve marks an open dispute resolved with an optional note (the simple
// accept-with-no-credit path). Kept for the existing callers/back-compat.
func (s *DisputeService) Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error {
	return s.repo.Resolve(ctx, tenantID, id, note)
}

// DisputeResolution captures an operator's decision on an open dispute.
type DisputeResolution struct {
	// Accept closes the dispute in the customer's favor (status resolved);
	// Accept=false rejects it (status rejected).
	Accept bool
	Note   string
	// IssueCredit (accept only) issues an adjustment credit note against the
	// disputed invoice. CreditAmount in minor units; <=0 defaults to the
	// invoice's amount still due (falling back to its total).
	IssueCredit  bool
	CreditAmount int64
}

// ResolveWithOutcome records an operator's decision on an open dispute. On
// accept it may issue an adjustment credit note against the disputed invoice —
// which reuses the fully-ledgered credit-note Create path, so no new money-path
// code is introduced here. The credit is issued BEFORE the dispute is closed:
// if it fails the dispute stays open for a clean retry, never a resolved
// dispute with no promised credit. Returns the created credit note (or nil).
func (s *DisputeService) ResolveWithOutcome(
	ctx context.Context,
	tenantID, actorID uuid.UUID,
	actorRole string,
	id uuid.UUID,
	res DisputeResolution,
) (*domain.CreditNote, error) {
	dispute, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load dispute: %w", err)
	}
	if dispute == nil || dispute.TenantID != tenantID {
		return nil, domain.ErrDisputeNotFound
	}
	if dispute.Status != domain.DisputeStatusOpen {
		return nil, domain.ErrDisputeNotFound
	}

	if !res.Accept {
		if err := s.repo.Close(ctx, tenantID, id, domain.DisputeStatusRejected, res.Note); err != nil {
			return nil, err
		}
		return nil, nil
	}

	var credit *domain.CreditNote
	if res.IssueCredit {
		if s.creditNotes == nil || s.invoices == nil {
			return nil, fmt.Errorf("credit issuance is not enabled")
		}
		inv, err := s.invoices.GetByIDPublic(ctx, dispute.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("load disputed invoice: %w", err)
		}
		if inv == nil || inv.TenantID != tenantID {
			return nil, fmt.Errorf("disputed invoice not found")
		}

		amount := res.CreditAmount
		if amount <= 0 {
			amount = inv.AmountDue
			if amount <= 0 {
				amount = inv.Total
			}
		}
		if amount > inv.Total {
			return nil, fmt.Errorf("credit amount exceeds the invoice total")
		}

		invoiceID := inv.ID
		credit, err = s.creditNotes.Create(ctx, tenantID, actorID, actorRole, domain.CreateCreditNoteRequest{
			CustomerID: dispute.CustomerID,
			InvoiceID:  &invoiceID,
			Amount:     amount,
			Currency:   inv.Currency,
			Reason:     "dispute_resolution",
			Type:       string(domain.CreditNoteTypeAdjustment),
		})
		if err != nil {
			return nil, fmt.Errorf("issue resolution credit: %w", err)
		}
	}

	if err := s.repo.Close(ctx, tenantID, id, domain.DisputeStatusResolved, res.Note); err != nil {
		return credit, err
	}
	return credit, nil
}
