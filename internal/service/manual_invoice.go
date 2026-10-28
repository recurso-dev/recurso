package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// Manual invoices: a one-off invoice whose lines, issuing entity and dates are
// supplied by the caller instead of derived from a plan, a subscription or a
// quote. Built for loading already-priced receivables (a billing-run export)
// into Recurso. Unlike a quote conversion it can carry negative (waiver /
// discount) lines, pick the issuing legal entity, and set the invoice and due
// dates. The invoice is persisted through the same repository path as every
// other invoice (gapless per-entity number, atomic line items) and posts its
// ledger leg exactly like a quote conversion (ADR-002 / ADR-004).

const (
	// maxManualInvoiceLines bounds one invoice; larger runs should be split.
	maxManualInvoiceLines = 500
	// maxManualLineAmount keeps every sum far inside int64 and rejects typos.
	maxManualLineAmount = int64(1_000_000_000_000) // 10 billion major units
	// maxManualDescriptionLen keeps a single PDF line sane.
	maxManualDescriptionLen = 500
	// maxManualExternalRefLen bounds the idempotency key.
	maxManualExternalRefLen = 200
	// manualDunningOwner keeps the dunning scheduler away from imported
	// receivables: it only claims invoices owned by "scheduler". Collecting an
	// imported balance is a human decision, not an automated email/retry.
	manualDunningOwner = "manual"
	// defaultManualPaymentTerms applies when neither a due date nor terms is given.
	defaultManualPaymentTerms = "net30"
)

// ErrInvalidManualInvoice marks a request the caller must fix (HTTP 400).
// Detail is carried in the wrapped message.
var ErrInvalidManualInvoice = errors.New("invalid manual invoice")

// ErrManualInvoiceRefConflict means the external_ref was already used for a
// DIFFERENT invoice (HTTP 409). Replaying a key is only a no-op when the
// content is identical; otherwise the caller changed something and must pick a
// new key (or fix the source) rather than have the change silently dropped.
var ErrManualInvoiceRefConflict = errors.New("external_ref already used with different content")

func invalidManual(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidManualInvoice, fmt.Sprintf(format, args...))
}

// ManualInvoiceLine is one itemized line. Amount is signed minor units: a
// negative line (waiver, discount) reduces the invoice total.
type ManualInvoiceLine struct {
	Description string
	Quantity    int   // 0 means 1
	UnitAmount  int64 // 0 means Amount / Quantity (must divide evenly)
	Amount      int64
}

// CreateManualInvoiceInput is the validated-on-entry request.
type CreateManualInvoiceInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	// EntityID is the issuing legal entity. Nil resolves to the tenant's primary.
	EntityID *uuid.UUID
	Currency string
	// InvoiceDate is the issue date shown on the invoice; zero means now.
	InvoiceDate time.Time
	// DueDate wins over PaymentTerms when set.
	DueDate      *time.Time
	PaymentTerms string // net0/net15/net30/net60…; empty → net30
	// ExternalRef is an optional idempotency key, unique per tenant. Replaying a
	// key returns the original invoice instead of creating a second one.
	ExternalRef string
	Lines       []ManualInvoiceLine
}

// ManualInvoiceService creates caller-specified invoices.
type ManualInvoiceService struct {
	invoiceRepo  *db.InvoiceRepository
	customerRepo port.CustomerRepository
	entityRepo   *db.EntityRepository
	txManager    *db.TxManager
	// ledger posts the invoice's AR → Revenue leg. nil-safe via SetLedgerPoster;
	// production MUST wire it or the reconciler flags every invoice.
	ledger invoiceLedgerPoster
	now    func() time.Time
}

func NewManualInvoiceService(
	invoiceRepo *db.InvoiceRepository,
	customerRepo port.CustomerRepository,
	entityRepo *db.EntityRepository,
	txManager *db.TxManager,
) *ManualInvoiceService {
	return &ManualInvoiceService{
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		entityRepo:   entityRepo,
		txManager:    txManager,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// SetLedgerPoster wires the ledger so every created invoice posts its
// double-entry leg, matching every other invoice-creating path.
func (s *ManualInvoiceService) SetLedgerPoster(ledger invoiceLedgerPoster) {
	s.ledger = ledger
}

// buildManualInvoice validates the input and assembles the invoice. Pure (no
// I/O) so every rule is unit-testable without a database.
func buildManualInvoice(in CreateManualInvoiceInput, now time.Time) (*domain.Invoice, error) {
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(currency) != 3 {
		return nil, invalidManual("currency must be a 3-letter code")
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return nil, invalidManual("currency must be a 3-letter code")
		}
	}
	if len(in.Lines) == 0 {
		return nil, invalidManual("at least one line is required")
	}
	if len(in.Lines) > maxManualInvoiceLines {
		return nil, invalidManual("too many lines (%d); the limit is %d per invoice", len(in.Lines), maxManualInvoiceLines)
	}
	if len(in.ExternalRef) > maxManualExternalRefLen {
		return nil, invalidManual("external_ref is longer than %d characters", maxManualExternalRefLen)
	}

	invoiceDate := in.InvoiceDate
	if invoiceDate.IsZero() {
		invoiceDate = now
	}
	terms := strings.TrimSpace(in.PaymentTerms)
	if terms == "" {
		terms = defaultManualPaymentTerms
	}
	dueDate := domain.CalculateDueDate(invoiceDate, terms)
	if in.DueDate != nil {
		dueDate = *in.DueDate
	}
	if dueDate.Before(invoiceDate) {
		return nil, invalidManual("due date is before the invoice date")
	}

	invID := uuid.New()
	lines := make([]domain.InvoiceItem, 0, len(in.Lines))
	var total int64
	for i, l := range in.Lines {
		n := i + 1
		desc := strings.TrimSpace(l.Description)
		if desc == "" {
			return nil, invalidManual("line %d: description is required", n)
		}
		if len(desc) > maxManualDescriptionLen {
			return nil, invalidManual("line %d: description is longer than %d characters", n, maxManualDescriptionLen)
		}
		if l.Amount > maxManualLineAmount || l.Amount < -maxManualLineAmount {
			return nil, invalidManual("line %d: amount is out of range", n)
		}
		qty := l.Quantity
		if qty < 0 {
			return nil, invalidManual("line %d: quantity cannot be negative", n)
		}
		if qty == 0 {
			qty = 1
		}
		unit := l.UnitAmount
		switch {
		case unit == 0:
			if l.Amount%int64(qty) != 0 {
				return nil, invalidManual("line %d: amount %d does not divide evenly by quantity %d; send unit_amount", n, l.Amount, qty)
			}
			unit = l.Amount / int64(qty)
		case unit*int64(qty) != l.Amount:
			return nil, invalidManual("line %d: quantity × unit_amount (%d) does not equal amount (%d)", n, unit*int64(qty), l.Amount)
		}
		// No tax: a manual invoice carries pre-agreed amounts. InvoiceTax{} keeps
		// the per-line tax columns at zero so Σ lines == subtotal == total.
		// Lines are read back ORDER BY created_at, id. Stamping each one a microsecond
		// after the previous keeps them in the order the caller sent (a waiver stays
		// next to its fee) instead of the random order of equal timestamps.
		lines = append(lines, newInvoiceLine(invID, desc, "", qty, unit, l.Amount, InvoiceTax{}, invoiceDate.Add(time.Duration(i)*time.Microsecond)))
		total += l.Amount
	}
	if total < 0 {
		return nil, invalidManual("net total is negative (%d); issue a credit note instead", total)
	}

	inv := &domain.Invoice{
		ID:               invID,
		TenantID:         in.TenantID,
		EntityID:         in.EntityID,
		CustomerID:       in.CustomerID,
		BillingReason:    domain.BillingReasonManual,
		Status:           domain.InvoiceStatusOpen,
		Currency:         currency,
		Subtotal:         total,
		Total:            total,
		AmountDue:        total,
		LineItems:        lines,
		CreatedAt:        invoiceDate,
		UpdatedAt:        now,
		DueDate:          dueDate,
		PaymentTerms:     terms,
		DunningManagedBy: manualDunningOwner,
	}
	// A fully-waived invoice owes nothing: record it as settled so it never
	// shows up as an open receivable or an overdue/dunning candidate. The
	// reconciler needs no payment leg for a paid invoice with amount_paid = 0.
	if total == 0 {
		paidAt := invoiceDate
		inv.Status = domain.InvoiceStatusPaid
		inv.PaidAt = &paidAt
	}
	return inv, nil
}

// CreateManualInvoice validates and persists a manual invoice. created is false
// when ExternalRef matched an existing invoice (idempotent replay); the
// original invoice is returned unchanged.
func (s *ManualInvoiceService) CreateManualInvoice(ctx context.Context, in CreateManualInvoiceInput) (inv *domain.Invoice, created bool, err error) {
	ctx = context.WithValue(ctx, domain.TenantIDKey, in.TenantID)

	built, err := buildManualInvoice(in, s.now())
	if err != nil {
		return nil, false, err
	}

	// Replay: the same key with the same content returns the original invoice.
	// Checked before live-state validation so a replay still succeeds after, say,
	// the customer was deactivated.
	if in.ExternalRef != "" {
		if existing, ok, rerr := s.loadByRef(ctx, in.TenantID, in.ExternalRef); rerr != nil {
			return nil, false, rerr
		} else if ok {
			if cerr := sameManualInvoice(existing, built); cerr != nil {
				return nil, false, cerr
			}
			return existing, false, nil
		}
	}

	customer, err := s.customerRepo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return nil, false, fmt.Errorf("load customer: %w", err)
	}
	if customer == nil { // not found, or another tenant's customer
		return nil, false, invalidManual("customer not found")
	}
	if in.EntityID != nil {
		entity, eerr := s.entityRepo.GetByID(ctx, *in.EntityID, in.TenantID)
		if eerr != nil {
			return nil, false, eerr
		}
		if entity == nil {
			return nil, false, invalidManual("entity not found")
		}
	}

	if err := s.txManager.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.invoiceRepo.CreateWithTx(ctx, tx, built); err != nil {
			return fmt.Errorf("create manual invoice: %w", err)
		}
		if in.ExternalRef != "" {
			return s.invoiceRepo.SetExternalRefWithTx(ctx, tx, in.TenantID, built.ID, in.ExternalRef)
		}
		return nil
	}); err != nil {
		// A concurrent request with the same key won the race: its commit made
		// our UPDATE violate the unique index. Replay it.
		if in.ExternalRef != "" && db.IsUniqueViolation(err) {
			if existing, ok, rerr := s.loadByRef(ctx, in.TenantID, in.ExternalRef); rerr == nil && ok {
				if cerr := sameManualInvoice(existing, built); cerr != nil {
					return nil, false, cerr
				}
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	// Post the AR → Revenue leg, like every other invoice-creating flow.
	// Best-effort + reconciliation, matching quote conversion: the invoice is
	// already durable, and a missing leg is reported by Finance → Reconciliation.
	if s.ledger != nil {
		if lerr := s.ledger.RecordInvoice(ctx, built); lerr != nil {
			slog.ErrorContext(ctx, "ledger write failed on manual invoice — needs reconciliation",
				"invoice_id", built.ID, "invoice_number", built.InvoiceNumber, "error", lerr)
		}
	}
	return built, true, nil
}

// sameManualInvoice reports whether a stored invoice is what the caller is
// asking for again: same customer, issuing entity, currency, net total and the
// same multiset of line amounts. Anything else is a conflicting reuse of the key.
func sameManualInvoice(existing, want *domain.Invoice) error {
	sameEntity := (existing.EntityID == nil && want.EntityID == nil) ||
		(existing.EntityID != nil && want.EntityID != nil && *existing.EntityID == *want.EntityID)
	if existing.CustomerID != want.CustomerID || !sameEntity ||
		!strings.EqualFold(existing.Currency, want.Currency) || existing.Total != want.Total {
		return fmt.Errorf("%w: invoice %s differs in customer, entity, currency or total", ErrManualInvoiceRefConflict, existing.InvoiceNumber)
	}
	amounts := func(items []domain.InvoiceItem) []int64 {
		out := make([]int64, len(items))
		for i, it := range items {
			out[i] = it.Amount
		}
		sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
		return out
	}
	a, b := amounts(existing.LineItems), amounts(want.LineItems)
	if len(a) != len(b) {
		return fmt.Errorf("%w: invoice %s has %d lines, this request has %d", ErrManualInvoiceRefConflict, existing.InvoiceNumber, len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("%w: invoice %s has different line amounts", ErrManualInvoiceRefConflict, existing.InvoiceNumber)
		}
	}
	return nil
}

func (s *ManualInvoiceService) loadByRef(ctx context.Context, tenantID uuid.UUID, ref string) (*domain.Invoice, bool, error) {
	id, found, err := s.invoiceRepo.GetIDByExternalRef(ctx, tenantID, ref)
	if err != nil || !found {
		return nil, false, err
	}
	inv, err := s.invoiceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, false, fmt.Errorf("load replayed invoice: %w", err)
	}
	return inv, true, nil
}
