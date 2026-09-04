package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Manual-collections-action errors. The handler maps these to HTTP status codes;
// each is a specific, user-facing reason an operator action was refused.
var (
	ErrCollectionInvoiceNotFound = errors.New("invoice not found")
	ErrRetryNotPastDue           = errors.New("only a past-due invoice can be retried")
	ErrRetryPaused               = errors.New("dunning is paused for this invoice")
	ErrRetryMandate              = errors.New("mandate auto-debit invoices cannot be manually retried")
	ErrRetryInFlight             = errors.New("a payment attempt is already in flight")
)

// collectionsActionRepo is the invoice-mutation surface the manual controls need.
// *db.InvoiceRepository satisfies it.
type collectionsActionRepo interface {
	GetRetryEligibility(ctx context.Context, tenantID, invoiceID uuid.UUID) (domain.InvoiceRetryEligibility, error)
	RequeueForRetry(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error)
	SetDunningPaused(ctx context.Context, tenantID, invoiceID uuid.UUID, paused bool) (bool, error)
	MarkUncollectibleScoped(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error)
}

// collectionsInFlightChecker reports whether an invoice has a settling payment
// attempt (ACH). *db.PaymentAttemptRepository satisfies it.
type collectionsInFlightChecker interface {
	HasInFlightForInvoice(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}

// CollectionsActionService is the operator-initiated collections mutations
// (Collections Intelligence Inc 3): retry-now, pause/resume dunning, and manual
// write-off. It posts NO ledger legs — retry defers to the existing worker
// settle path, and uncollectible is a status (matching the automated path).
type CollectionsActionService struct {
	repo     collectionsActionRepo
	attempts collectionsInFlightChecker // nil-safe
	logger   *slog.Logger
	// Write-off ledger reversal (nil-safe): without these a write-off is a
	// status flip that leaves AR and Deferred overstated forever.
	ledger   writeOffLedgerPoster
	invoices writeOffInvoiceReader
}

// writeOffLedgerPoster is satisfied by *LedgerService.
type writeOffLedgerPoster interface {
	RecordInvoiceWriteOff(ctx context.Context, invoice *domain.Invoice) error
}

// writeOffInvoiceReader is satisfied by *db.InvoiceRepository.
type writeOffInvoiceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
}

// SetWriteOffLedger wires the ledger reversal posted on write-off. Nil-safe.
func (s *CollectionsActionService) SetWriteOffLedger(l writeOffLedgerPoster, r writeOffInvoiceReader) {
	s.ledger = l
	s.invoices = r
}

func NewCollectionsActionService(repo collectionsActionRepo) *CollectionsActionService {
	return &CollectionsActionService{repo: repo, logger: slog.Default().With("service", "collections_actions")}
}

// SetInFlightChecker wires the ACH in-flight guard so retry-now can't stack a
// charge on top of a still-settling attempt. Nil-safe.
func (s *CollectionsActionService) SetInFlightChecker(c collectionsInFlightChecker) { s.attempts = c }

// RetryNow requeues a failing invoice for an immediate worker retry. It refuses
// (with a precise error) anything that isn't a tenant-owned, past-due, un-paused,
// non-mandate invoice with no in-flight attempt — the same safety envelope the
// automated engine honors, so a manual click can never double-charge.
func (s *CollectionsActionService) RetryNow(ctx context.Context, tenantID, invoiceID uuid.UUID) error {
	elig, err := s.repo.GetRetryEligibility(ctx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if !elig.Found {
		return ErrCollectionInvoiceNotFound
	}
	if elig.Status != string(domain.InvoiceStatusPastDue) {
		return ErrRetryNotPastDue
	}
	if elig.Paused {
		return ErrRetryPaused
	}
	if elig.IsMandate {
		return ErrRetryMandate
	}
	if s.attempts != nil {
		inFlight, err := s.attempts.HasInFlightForInvoice(ctx, invoiceID)
		if err != nil {
			return err
		}
		if inFlight {
			return ErrRetryInFlight
		}
	}

	ok, err := s.repo.RequeueForRetry(ctx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if !ok {
		// The atomic WHERE rejected it — state changed between the read and the
		// write (e.g. just settled, paused, or marked uncollectible). Report as
		// not-retryable rather than a false success.
		return ErrRetryNotPastDue
	}
	s.logger.InfoContext(ctx, "manual retry-now requeued invoice", "invoice_id", invoiceID, "tenant_id", tenantID)
	return nil
}

// SetPaused pauses or resumes automated dunning on an invoice.
func (s *CollectionsActionService) SetPaused(ctx context.Context, tenantID, invoiceID uuid.UUID, paused bool) error {
	ok, err := s.repo.SetDunningPaused(ctx, tenantID, invoiceID, paused)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCollectionInvoiceNotFound
	}
	s.logger.InfoContext(ctx, "dunning pause toggled", "invoice_id", invoiceID, "paused", paused, "tenant_id", tenantID)
	return nil
}

// MarkUncollectible is the operator-initiated write-off: the status flip plus
// the ledger reversal (DR Deferred/Revenue + Tax Payable, CR AR) so the books
// stop carrying money that will never arrive.
func (s *CollectionsActionService) MarkUncollectible(ctx context.Context, tenantID, invoiceID uuid.UUID) error {
	ok, err := s.repo.MarkUncollectibleScoped(ctx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCollectionInvoiceNotFound
	}
	if s.ledger != nil && s.invoices != nil {
		inv, ierr := s.invoices.GetByID(ctx, invoiceID)
		if ierr != nil || inv == nil || inv.TenantID != tenantID {
			s.logger.ErrorContext(ctx, "write-off ledger reversal skipped: invoice lookup failed",
				"invoice_id", invoiceID, "error", ierr)
		} else if lerr := s.ledger.RecordInvoiceWriteOff(ctx, inv); lerr != nil {
			// Surface loudly for reconciliation; the status change stands — the
			// tie-out's unscheduled bucket keeps un-reversed write-offs visible.
			s.logger.ErrorContext(ctx, "write-off ledger reversal failed", "invoice_id", invoiceID, "error", lerr)
		}
	}
	s.logger.InfoContext(ctx, "invoice manually marked uncollectible", "invoice_id", invoiceID, "tenant_id", tenantID)
	return nil
}
