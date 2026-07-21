package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// EU e-invoice delivery retry schedule. A transmission to the Access Point can
// fail transiently; the background worker redrives on this exponential backoff
// and gives up after MaxEUEInvoiceRetries. Kept here (not in the worker package)
// so GenerateForInvoice can schedule the first attempt from the same source.
var EUEInvoiceBackoff = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	1 * time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

// MaxEUEInvoiceRetries is the number of failed delivery attempts after which a
// record is left permanently failed (next_retry_at cleared) for manual attention.
const MaxEUEInvoiceRetries = 5

// EUEInvoiceService orchestrates EU e-invoice generation for an invoice: it gates
// on the tenant's opt-in, projects the seller from the tenant EU config and the
// buyer from the customer, generates the EN 16931 UBL document, hands it to the
// transport, and persists the document + status. Best-effort and nil-safe — a
// tenant that hasn't opted in is a silent no-op.
type EUEInvoiceService struct {
	configRepo euConfigReader
	invoices   euInvoiceWriter
	transport  port.EUInvoiceTransport
	logger     *slog.Logger
}

// euConfigReader / euInvoiceWriter are the narrow persistence the service needs;
// satisfied by *db.TenantEUConfigRepository and *db.EUInvoiceRepository.
type euConfigReader interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantEUConfig, error)
}

type euInvoiceWriter interface {
	Upsert(ctx context.Context, e *domain.EUInvoice) error
}

func NewEUEInvoiceService(configRepo euConfigReader, invoices euInvoiceWriter, transport port.EUInvoiceTransport) *EUEInvoiceService {
	return &EUEInvoiceService{
		configRepo: configRepo,
		invoices:   invoices,
		transport:  transport,
		logger:     slog.Default().With("service", "eu_einvoice"),
	}
}

// GenerateForInvoice generates + transmits the EU e-invoice for a committed
// invoice when the tenant has opted in. Returns (nil, nil) when EU e-invoicing
// is off for the tenant. A generation/transmission failure is recorded on the
// stored record (status=failed) and returned, but never rolls back the invoice.
func (s *EUEInvoiceService) GenerateForInvoice(ctx context.Context, inv *domain.Invoice, customer *domain.Customer) (*domain.EUInvoice, error) {
	if s == nil || s.transport == nil || s.configRepo == nil || s.invoices == nil {
		return nil, nil
	}
	cfg, err := s.configRepo.GetByTenantID(ctx, inv.TenantID)
	if err != nil {
		return nil, fmt.Errorf("eu e-invoice: load tenant config: %w", err)
	}
	if cfg == nil || !cfg.Enabled {
		return nil, nil // not opted in — silent no-op
	}

	seller := cfg.SellerParty()
	buyer := buildEUBuyer(customer)
	rec := &domain.EUInvoice{
		TenantID:       inv.TenantID,
		InvoiceID:      inv.ID,
		Syntax:         domain.EUInvoiceSyntaxUBL,
		RecipientVATID: buyer.VATID,
	}

	doc, err := BuildUBLInvoice(inv, seller, buyer)
	if err != nil {
		// Generation failure is a data problem (missing VAT id, invalid country)
		// that won't fix itself: mark failed but leave next_retry_at nil so the
		// worker never redrives it — it surfaces for manual correction.
		rec.Status = domain.EUInvoiceStatusFailed
		rec.ErrorMessage = err.Error()
		_ = s.invoices.Upsert(ctx, rec)
		s.logger.Warn("eu e-invoice generation failed", "invoice_id", inv.ID, "error", err)
		return rec, err
	}
	rec.Document = string(doc)
	rec.Status = domain.EUInvoiceStatusGenerated

	res, terr := s.transport.Transmit(ctx, domain.EUInvoiceSyntaxUBL, buyer.VATID, doc)
	if terr != nil || res == nil {
		// Transmission failure is the retriable case: the document is built and
		// stored, so mark it failed and schedule the first redrive. The worker
		// re-transmits (never regenerates) on the backoff schedule.
		rec.Status = domain.EUInvoiceStatusFailed
		rec.ErrorMessage = fmt.Sprintf("transmit failed: %v", terr)
		next := time.Now().UTC().Add(EUEInvoiceBackoff[0])
		rec.NextRetryAt = &next
		if err := s.invoices.Upsert(ctx, rec); err != nil {
			s.logger.Error("eu e-invoice: persist after transmit failure", "invoice_id", inv.ID, "error", err)
		}
		s.logger.Warn("eu e-invoice transmit failed; document stored and scheduled for retry", "invoice_id", inv.ID, "error", terr, "next_retry_at", next)
		return rec, terr
	}
	rec.Status = res.Status
	rec.MessageID = res.MessageID
	if err := s.invoices.Upsert(ctx, rec); err != nil {
		return rec, fmt.Errorf("eu e-invoice: persist: %w", err)
	}
	s.logger.Info("eu e-invoice generated and sent", "invoice_id", inv.ID, "message_id", rec.MessageID)
	return rec, nil
}

// RetryTransmission re-hands a stored document to the transport. The document is
// generated once and immutable, so a delivery redrive only needs to re-transmit
// it — no regeneration, hence no dependency on the invoice/customer. Returns the
// transport outcome; the caller (retry worker) owns the backoff/count bookkeeping.
func (s *EUEInvoiceService) RetryTransmission(ctx context.Context, rec *domain.EUInvoice) (*domain.EUInvoiceTransmission, error) {
	if s == nil || s.transport == nil {
		return nil, errors.New("eu e-invoice: no transport configured")
	}
	if rec == nil || rec.Document == "" {
		return nil, errors.New("eu e-invoice: no stored document to re-transmit")
	}
	syntax := rec.Syntax
	if syntax == "" {
		syntax = domain.EUInvoiceSyntaxUBL
	}
	res, err := s.transport.Transmit(ctx, syntax, rec.RecipientVATID, []byte(rec.Document))
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("eu e-invoice: transport returned no result")
	}
	return res, nil
}

// buildEUBuyer projects a customer into the EN 16931 buyer party. The VAT id is
// the customer's tax id; the country is normalized to an ISO 3166-1 alpha-2 code
// (BuildUBLInvoice rejects a missing/invalid one, which surfaces the data gap
// rather than emitting a non-compliant document).
func buildEUBuyer(c *domain.Customer) domain.EUParty {
	if c == nil {
		return domain.EUParty{}
	}
	name := domain.PtrToString(c.Name)
	if name == "" {
		name = c.Email
	}
	return domain.EUParty{
		Name:        name,
		VATID:       domain.PtrToString(c.TaxID),
		CountryCode: euCountryISO2(c.BillingAddress.Country),
		Street:      c.BillingAddress.Line1,
		City:        c.BillingAddress.City,
		PostalZone:  c.BillingAddress.Zip,
	}
}

// euCountryISO2 normalizes a country to its ISO 3166-1 alpha-2 code: an existing
// 2-letter code is upper-cased; a small set of common EU country names is
// mapped; anything else passes through (validation catches an invalid value).
func euCountryISO2(country string) string {
	s := strings.TrimSpace(country)
	if len(s) == 2 {
		return strings.ToUpper(s)
	}
	if code, ok := euCountryNames[strings.ToLower(s)]; ok {
		return code
	}
	return s
}

var euCountryNames = map[string]string{
	"austria": "AT", "belgium": "BE", "bulgaria": "BG", "croatia": "HR", "cyprus": "CY",
	"czechia": "CZ", "czech republic": "CZ", "denmark": "DK", "estonia": "EE", "finland": "FI",
	"france": "FR", "germany": "DE", "greece": "GR", "hungary": "HU", "ireland": "IE",
	"italy": "IT", "latvia": "LV", "lithuania": "LT", "luxembourg": "LU", "malta": "MT",
	"netherlands": "NL", "poland": "PL", "portugal": "PT", "romania": "RO", "slovakia": "SK",
	"slovenia": "SI", "spain": "ES", "sweden": "SE",
}
