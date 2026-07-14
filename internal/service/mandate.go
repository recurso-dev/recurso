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

	// ENG-153: apply the customer's account credit against this debit. Preview
	// the available adjustment-credit balance and charge the gateway only the
	// net; the actual draw-down happens after the invoice exists.
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
	var err error

	// Charge the gateway for the net. If credit fully covers the debit, skip the
	// gateway entirely (nothing to collect).
	var result *port.PaymentResult
	if netCharge > 0 {
		result, err = s.gateway.ExecuteMandateDebit(ctx, port.MandateDebitRequest{
			TokenID:            mandate.RazorpayTokenID,
			RazorpayCustomerID: mandate.RazorpayCustomerID,
			Email:              customer.Email,
			Contact:            customer.Phone,
			Amount:             netCharge,
			Currency:           currency,
			InvoiceID:          invoiceID.String(),
			// Stable per billing cycle: LastDebitAt only advances when a cycle is
			// SUCCESSFULLY debited, so every retry of the current cycle carries the
			// same key and the gateway dedupes the charge (ENG-190). This bounds the
			// exposure where a debit succeeds at the gateway but a later local step
			// fails and the scheduler re-attempts the same cycle.
			IdempotencyKey: mandateDebitIdempotencyKey(mandate),
		})
		if err != nil {
			return fmt.Errorf("mandate debit failed: %w", err)
		}
		if !result.Success {
			return fmt.Errorf("mandate debit unsuccessful: %s", result.ErrorMsg)
		}
	}

	// Create the invoice OPEN, not paid. The recurring debit captures
	// asynchronously, so the invoice is settled ONLY by the order.paid /
	// payment.captured webhook (which resolves it via notes.invoice_id). Booking
	// it paid here would record revenue that was never collected (ENG-141). The
	// invoice total stays GROSS; credit application (below) records credit_applied.
	now := time.Now()
	invoice := &domain.Invoice{
		ID:             invoiceID,
		TenantID:       mandate.TenantID,
		CustomerID:     mandate.CustomerID,
		SubscriptionID: mandate.SubscriptionID,
		InvoiceNumber:  fmt.Sprintf("MD-%s", invoiceID.String()[:8]),
		BillingReason:  "mandate_debit",
		AmountDue:      total,
		AmountPaid:     0,
		Currency:       currency,
		Subtotal:       subtotal,
		TaxAmount:      tax.Total,
		Total:          total,
		IGSTAmount:     tax.IGST,
		CGSTAmount:     tax.CGST,
		SGSTAmount:     tax.SGST,
		HSNCode:        tax.HSN,
		Status:         domain.InvoiceStatusOpen,
		// Single line for the debit, carrying the resolved GST split (zero split on
		// the legacy fixed-amount path).
		LineItems: []domain.InvoiceItem{
			newInvoiceLine(invoiceID, "Mandate debit", tax.HSN, 1, subtotal, subtotal, tax, time.Time{}),
		},
		CreatedAt: now,
		DueDate:   now,
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return fmt.Errorf("failed to create invoice for mandate debit: %w", err)
	}

	// Draw down the credit against the now-persisted invoice. When it fully
	// covers the debit, ApplyAdjustmentCredits marks the invoice paid (no webhook
	// will arrive, since we skipped the gateway). Best-effort: a failure leaves
	// the invoice open at full amount for the webhook/reconciliation to settle.
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

	// Capture the gateway payment id when the debit response carries a real one
	// (pay_*) — refunds are issued against it. The recurring-charge call returns
	// the pay_* id; the order.paid webhook also records it via SetGatewayPaymentID
	// (see WebhookHandler.HandleRazorpay), so this is a best-effort early capture.
	// Only a payment id is stored — an order id (order_*) would poison
	// gateway_payment_id and break refunds, so isGatewayPaymentID guards it.
	if result != nil && isGatewayPaymentID(result.PaymentID) {
		invoice.GatewayPaymentID = result.PaymentID
		if err := s.invoiceRepo.SetGatewayPaymentID(ctx, invoice.ID, result.PaymentID); err != nil {
			// The debit succeeded and the invoice exists; failing the whole
			// debit here would re-run it next cycle and double-charge. Refunds
			// for this invoice fall back to manual_required instead.
			slog.Default().Error("failed to record gateway payment id for mandate debit",
				"invoice_id", invoice.ID, "payment_id", result.PaymentID, "error", err)
		}
	}

	// Advance mandate schedule
	mandate.LastDebitAt = &now
	mandate.PreDebitNotified = false
	nextDebit := s.calculateNextDebit(now, mandate.Frequency)
	mandate.NextDebitAt = &nextDebit

	return s.mandateRepo.Update(ctx, mandate)
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
