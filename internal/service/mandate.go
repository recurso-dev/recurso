package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type MandateService struct {
	mandateRepo  port.MandateRepository
	gateway      port.PaymentGateway
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
	// creditApplier applies adjustment credit-note balances to the mandate-debit
	// invoice, reducing the amount actually charged (ENG-153). nil-safe.
	creditApplier creditApplier
	// Billing resolution for DebitSubscription: the subscription's plan price and
	// its resolved tax become the charge, instead of debiting the mandate ceiling
	// (ENG-165). All three are set together via SetBillingResolver; nil until then.
	subscriptionRepo port.SubscriptionRepository
	planRepo         port.PlanRepository
	taxResolver      mandateTaxResolver
}

// mandateTaxResolver is the slice of TaxResolver the mandate debit needs: resolve
// the tax for one amount against a customer's jurisdiction. *TaxResolver satisfies it.
type mandateTaxResolver interface {
	ResolveInvoiceTax(ctx context.Context, tenantID uuid.UUID, customer *domain.Customer, currency string, amount int64, hsn string) InvoiceTax
}

// SetCreditApplier wires credit application into the mandate-debit charge path (ENG-153).
func (s *MandateService) SetCreditApplier(a creditApplier) { s.creditApplier = a }

// SetBillingResolver wires the subscription/plan/tax lookups DebitSubscription
// uses to charge a mandate its subscription's real recurring amount (plan price
// + tax) rather than the authorized ceiling. Without it, DebitSubscription
// errors instead of guessing an amount (ENG-165).
func (s *MandateService) SetBillingResolver(subscriptionRepo port.SubscriptionRepository, planRepo port.PlanRepository, taxResolver mandateTaxResolver) {
	s.subscriptionRepo = subscriptionRepo
	s.planRepo = planRepo
	s.taxResolver = taxResolver
}

func NewMandateService(
	mandateRepo port.MandateRepository,
	gateway port.PaymentGateway,
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
) *MandateService {
	return &MandateService{
		mandateRepo:  mandateRepo,
		gateway:      gateway,
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
	}
}

// ErrCustomerPhoneRequired is returned when a UPI mandate is requested for a
// customer without a contact number — Razorpay rejects recurring registration
// links without one.
var ErrCustomerPhoneRequired = errors.New("customer phone number is required for a UPI mandate")

type CreateMandateInput struct {
	TenantID       uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	VPA            string
	MaxAmount      int64
	Frequency      string
}

type CreateMandateOutput struct {
	Mandate *domain.Mandate `json:"mandate"`
	AuthURL string          `json:"auth_url,omitempty"`
}

func (s *MandateService) CreateMandate(ctx context.Context, input CreateMandateInput) (*CreateMandateOutput, error) {
	customer, err := s.customerRepo.GetByID(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Razorpay requires a contact number on recurring registration links —
	// fail with a typed error so handlers can explain what's missing.
	if customer.Phone == "" {
		return nil, ErrCustomerPhoneRequired
	}

	result, err := s.gateway.CreateMandate(ctx, customer.Email, customer.Phone, input.VPA, input.MaxAmount, input.Frequency)
	if err != nil {
		return nil, fmt.Errorf("failed to create mandate with gateway: %w", err)
	}

	now := time.Now()
	mandate := &domain.Mandate{
		ID:                     uuid.New(),
		TenantID:               input.TenantID,
		CustomerID:             input.CustomerID,
		SubscriptionID:         input.SubscriptionID,
		MandateType:            "recurring",
		PaymentMethod:          "upi",
		VPA:                    input.VPA,
		RazorpayTokenID:        result.TokenID,
		RazorpaySubscriptionID: result.SubscriptionID,
		RazorpayCustomerID:     result.CustomerID,
		MaxAmount:              input.MaxAmount,
		Frequency:              input.Frequency,
		Status:                 domain.MandateStatusCreated,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := s.mandateRepo.Create(ctx, mandate); err != nil {
		return nil, fmt.Errorf("failed to save mandate: %w", err)
	}

	return &CreateMandateOutput{
		Mandate: mandate,
		AuthURL: result.AuthURL,
	}, nil
}

// HandleAuthorization activates a mandate when the gateway confirms the token.
// razorpayCustomerID may be empty; when present it is persisted so the token
// can later be revoked via Razorpay's customer-scoped token deletion API.
func (s *MandateService) HandleAuthorization(ctx context.Context, tokenID, razorpayCustomerID string) error {
	mandate, err := s.mandateRepo.GetByRazorpayTokenID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("mandate not found for token %s: %w", tokenID, err)
	}

	now := time.Now()
	mandate.Status = domain.MandateStatusActive
	mandate.AuthorizedAt = &now
	mandate.ActivatedAt = &now
	if razorpayCustomerID != "" {
		mandate.RazorpayCustomerID = razorpayCustomerID
	}

	return s.mandateRepo.Update(ctx, mandate)
}

// ExecuteDebit charges a fixed amount with no tax split. It is the low-level
// primitive (and legacy entry point) — the recurring scheduler uses
// DebitSubscription, which resolves the real amount. `amount` is treated as the
// full gross charge (subtotal == total, no GST split).
func (s *MandateService) ExecuteDebit(ctx context.Context, mandate *domain.Mandate, amount int64, currency string) error {
	// The mandate-debit scheduler calls this with a background context, but every
	// repo below is tenant-scoped (customer read, invoice create) and fails closed
	// on a missing tenant — inject the mandate's own tenant so the debit actually
	// runs instead of erroring at the customer lookup (tenant-context bug class).
	ctx = context.WithValue(ctx, domain.TenantIDKey, mandate.TenantID)

	customer, err := s.customerRepo.GetByID(ctx, mandate.CustomerID)
	if err != nil {
		return fmt.Errorf("mandate debit: load customer %s: %w", mandate.CustomerID, err)
	}
	return s.chargeMandate(ctx, mandate, customer, amount, InvoiceTax{}, currency)
}

// DebitSubscription charges the mandate's subscription its actual recurring
// amount — the plan's recurring price plus resolved tax — and stamps the invoice
// with the real CGST/SGST/IGST split. It replaces debiting mandate.MaxAmount
// directly, which is the authorization *ceiling* (set to ~2× the largest
// invoice) and over-charged roughly 2× every cycle (ENG-165).
func (s *MandateService) DebitSubscription(ctx context.Context, mandate *domain.Mandate) error {
	ctx = context.WithValue(ctx, domain.TenantIDKey, mandate.TenantID)

	if s.subscriptionRepo == nil || s.planRepo == nil || s.taxResolver == nil {
		return fmt.Errorf("mandate debit: billing resolver not configured")
	}
	if mandate.SubscriptionID == nil {
		return fmt.Errorf("mandate debit: mandate %s has no subscription to bill", mandate.ID)
	}

	sub, err := s.subscriptionRepo.GetByID(ctx, *mandate.SubscriptionID)
	if err != nil {
		return fmt.Errorf("mandate debit: load subscription %s: %w", *mandate.SubscriptionID, err)
	}
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return fmt.Errorf("mandate debit: load plan %s: %w", sub.PlanID, err)
	}
	price, ok := recurringPrice(plan)
	if !ok {
		return fmt.Errorf("mandate debit: plan %s has no recurring price", plan.ID)
	}

	customer, err := s.customerRepo.GetByID(ctx, mandate.CustomerID)
	if err != nil {
		return fmt.Errorf("mandate debit: load customer %s: %w", mandate.CustomerID, err)
	}

	tax := s.taxResolver.ResolveInvoiceTax(ctx, mandate.TenantID, customer, price.Currency, price.Amount, plan.HSNCode)
	total := price.Amount + tax.Total

	// Safety ceiling: the mandate authorizes debits only up to MaxAmount. A charge
	// above it means the plan/tax grew beyond what the customer authorized — skip
	// and surface it rather than over-charge (the original bug) or silently
	// under-charge by capping.
	if total > mandate.MaxAmount {
		return fmt.Errorf("mandate debit: computed charge %d exceeds authorized max %d for mandate %s", total, mandate.MaxAmount, mandate.ID)
	}

	return s.chargeMandate(ctx, mandate, customer, price.Amount, tax, price.Currency)
}

// chargeMandate is the shared debit core: charge the gateway the net of account
// credit, record an OPEN invoice with the given subtotal + tax split, apply
// credit, and advance the mandate schedule. ctx must already carry the tenant.
func (s *MandateService) chargeMandate(ctx context.Context, mandate *domain.Mandate, customer *domain.Customer, subtotal int64, tax InvoiceTax, currency string) error {
	invoiceID := uuid.New()
	total := subtotal + tax.Total
	now := time.Now()

	// The per-cycle claim key, captured BEFORE the schedule advances (which would
	// change it). The OPEN invoice carries it under a UNIQUE index, making the
	// invoice — not the gateway — the durable at-most-once claim (Razorpay does
	// not honor the idempotency header; verified for ENG-164).
	cycleKey := mandateDebitIdempotencyKey(mandate)

	// ENG-153: preview the customer's account credit so we charge the gateway only
	// the net; the actual draw-down happens after the invoice exists.
	var previewCredit int64
	if s.creditApplier != nil {
		if sum, sErr := s.creditApplier.SumApplicableAdjustments(ctx, mandate.TenantID, mandate.CustomerID, currency); sErr != nil {
			slog.Default().Error("mandate debit: credit preview failed; charging full amount",
				"customer_id", mandate.CustomerID, "error", sErr)
		} else {
			previewCredit = sum
			if previewCredit > total {
				previewCredit = total
			}
		}
	}
	netCharge := total - previewCredit

	// CLAIM FIRST — before any charge. Create the invoice OPEN with the per-cycle
	// key. If a prior attempt of this cycle already created it, the UNIQUE index
	// rejects this insert: the cycle is already claimed, so we must NOT charge
	// again — ensure the schedule has moved on and return cleanly. The invoice is
	// created OPEN and settled by the order.paid webhook (booking it paid here
	// would record revenue never collected — ENG-141).
	invoice := &domain.Invoice{
		ID:              invoiceID,
		TenantID:        mandate.TenantID,
		CustomerID:      mandate.CustomerID,
		SubscriptionID:  mandate.SubscriptionID,
		InvoiceNumber:   fmt.Sprintf("MD-%s", invoiceID.String()[:8]),
		BillingReason:   domain.BillingReasonMandateDebit,
		MandateCycleKey: cycleKey,
		AmountDue:       total,
		AmountPaid:      0,
		Currency:        currency,
		Subtotal:        subtotal,
		TaxAmount:       tax.Total,
		Total:           total,
		IGSTAmount:      tax.IGST,
		CGSTAmount:      tax.CGST,
		SGSTAmount:      tax.SGST,
		HSNCode:         tax.HSN,
		Status:          domain.InvoiceStatusOpen,
		// Single line for the debit, carrying the resolved GST split (zero split on
		// the legacy fixed-amount path).
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(invoiceID, "Mandate debit", tax.HSN, 1, subtotal, subtotal, tax, time.Time{}),
		},
		CreatedAt: now,
		DueDate:   now,
	}
	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		if isMandateCycleConflict(err) {
			slog.Default().Warn("mandate debit: cycle already claimed; skipping duplicate charge",
				"mandate_id", mandate.ID, "cycle_key", cycleKey)
			// A prior attempt already claimed (and possibly charged) this cycle.
			// Never charge again; just make sure the schedule advanced so we stop
			// re-claiming it.
			if aErr := s.advanceMandateSchedule(ctx, mandate, now); aErr != nil {
				slog.Default().Error("mandate debit: advance after cycle conflict failed",
					"mandate_id", mandate.ID, "error", aErr)
			}
			return nil
		}
		return fmt.Errorf("failed to create invoice for mandate debit: %w", err)
	}

	// Advance the schedule to the next full cycle NOW — before the charge — so a
	// charge that succeeds-then-crashes, or fails, can never re-run this cycle.
	if err := s.advanceMandateSchedule(ctx, mandate, now); err != nil {
		// The invoice (claim) exists, so a re-claim conflicts and skips the charge;
		// surface the error but the double-charge window stays closed.
		return fmt.Errorf("mandate debit: advance schedule: %w", err)
	}

	// Now charge the gateway for the net. If credit fully covers the debit, skip
	// the gateway entirely (nothing to collect).
	var result *port.PaymentResult
	if netCharge > 0 {
		var err error
		result, err = s.gateway.ExecuteMandateDebit(ctx, port.MandateDebitRequest{
			TokenID:            mandate.RazorpayTokenID,
			RazorpayCustomerID: mandate.RazorpayCustomerID,
			Email:              customer.Email,
			Contact:            customer.Phone,
			Amount:             netCharge,
			Currency:           currency,
			InvoiceID:          invoiceID.String(),
			// Sent for completeness; Razorpay does not dedupe on it (verified for
			// ENG-164), so the real guard is the claimed invoice above.
			IdempotencyKey: cycleKey,
		})
		if err != nil {
			// Cycle claimed + schedule advanced, so this won't re-charge; the OPEN
			// invoice is left for dunning / the next cycle.
			return fmt.Errorf("mandate debit failed: %w", err)
		}
		if !result.Success {
			return fmt.Errorf("mandate debit unsuccessful: %s", result.ErrorMsg)
		}
	}

	// Draw down the credit against the persisted invoice. When it fully covers the
	// debit, ApplyAdjustmentCredits marks the invoice paid (no webhook will arrive,
	// since we skipped the gateway). Best-effort.
	if s.creditApplier != nil && previewCredit > 0 {
		applied, aErr := s.creditApplier.ApplyAdjustmentCredits(ctx, mandate.TenantID, mandate.CustomerID, currency, invoice.ID, total)
		if aErr != nil {
			slog.Default().Error("mandate debit: credit application failed", "invoice_id", invoice.ID, "error", aErr)
		} else {
			invoice.CreditApplied = applied
			if applied != previewCredit {
				slog.Default().Warn("mandate debit: applied credit differs from preview (concurrent draw-down)",
					"invoice_id", invoice.ID, "preview", previewCredit, "applied", applied, "charged", netCharge)
			}
		}
	}

	// Best-effort early capture of the gateway payment id (pay_*) for refunds; the
	// order.paid webhook also records it. An order id (order_*) must never land in
	// gateway_payment_id, so isGatewayPaymentID guards it.
	if result != nil && isGatewayPaymentID(result.PaymentID) {
		invoice.GatewayPaymentID = result.PaymentID
		if err := s.invoiceRepo.SetGatewayPaymentID(ctx, invoice.ID, result.PaymentID); err != nil {
			slog.Default().Error("failed to record gateway payment id for mandate debit",
				"invoice_id", invoice.ID, "payment_id", result.PaymentID, "error", err)
		}
	}

	return nil
}

// advanceMandateSchedule moves the mandate to its next full billing cycle. Called
// before the gateway charge so a charge that fails or crashes can't re-run the
// same cycle (ENG-164).
func (s *MandateService) advanceMandateSchedule(ctx context.Context, mandate *domain.Mandate, now time.Time) error {
	mandate.LastDebitAt = &now
	mandate.PreDebitNotified = false
	nextDebit := s.calculateNextDebit(now, mandate.Frequency)
	mandate.NextDebitAt = &nextDebit
	return s.mandateRepo.Update(ctx, mandate)
}

// isMandateCycleConflict reports whether err is the UNIQUE-violation on the
// mandate cycle key — i.e. this billing cycle was already claimed by a prior
// (possibly crashed) debit attempt, so it must not be charged again.
func isMandateCycleConflict(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "mandate_cycle_key")
	}
	return false
}

// isGatewayPaymentID reports whether id is a refundable gateway payment
// identifier (Razorpay pay_*, Stripe pi_*/ch_*) rather than an order id —
// the Refund APIs of both gateways accept only payment identifiers.
// mandateDebitIdempotencyKey returns a key that is STABLE across every retry of
// the current billing cycle. LastDebitAt is set only when a cycle is
// successfully debited, so it uniquely identifies "the cycle after the last
// success" and doesn't change while that cycle is being (re)attempted — unlike
// NextDebitAt, which the claim lease rewrites on every tick. First-ever debit
// (LastDebitAt nil) keys on 0.
func mandateDebitIdempotencyKey(mandate *domain.Mandate) string {
	var last int64
	if mandate.LastDebitAt != nil {
		last = mandate.LastDebitAt.Unix()
	}
	return fmt.Sprintf("md-%s-%d", mandate.ID, last)
}

func isGatewayPaymentID(id string) bool {
	return strings.HasPrefix(id, "pay_") || strings.HasPrefix(id, "pi_") || strings.HasPrefix(id, "ch_")
}

func (s *MandateService) Revoke(ctx context.Context, mandateID, tenantID uuid.UUID) error {
	mandate, err := s.mandateRepo.GetByID(ctx, mandateID, tenantID)
	if err != nil {
		return fmt.Errorf("mandate not found: %w", err)
	}

	if mandate.RazorpayTokenID != "" {
		if err := s.gateway.RevokeMandate(ctx, mandate.RazorpayCustomerID, mandate.RazorpayTokenID); err != nil {
			return fmt.Errorf("failed to revoke mandate at gateway: %w", err)
		}
	}

	now := time.Now()
	mandate.Status = domain.MandateStatusRevoked
	mandate.RevokedAt = &now

	return s.mandateRepo.Update(ctx, mandate)
}

func (s *MandateService) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Mandate, error) {
	return s.mandateRepo.GetByID(ctx, id, tenantID)
}

func (s *MandateService) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Mandate, error) {
	return s.mandateRepo.List(ctx, tenantID)
}

// recurringPrice picks the plan's recurring price (the one billed each cycle),
// falling back to the first price if none is explicitly typed "recurring".
func recurringPrice(plan *domain.Plan) (domain.Price, bool) {
	for _, p := range plan.Prices {
		if p.Type == "recurring" {
			return p, true
		}
	}
	if len(plan.Prices) > 0 {
		return plan.Prices[0], true
	}
	return domain.Price{}, false
}

func (s *MandateService) calculateNextDebit(from time.Time, frequency string) time.Time {
	switch frequency {
	case "weekly":
		return from.AddDate(0, 0, 7)
	case "monthly":
		return from.AddDate(0, 1, 0)
	case "quarterly":
		return from.AddDate(0, 3, 0)
	case "yearly":
		return from.AddDate(1, 0, 0)
	default:
		return from.AddDate(0, 1, 0)
	}
}
