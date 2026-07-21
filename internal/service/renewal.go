package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// RenewalService is the billing-cycle engine (Lago-parity A1): it turns a
// claimed, period-ended subscription into a renewal invoice (flat fee in
// advance + metered usage in arrears via InvoiceService.GenerateInvoice),
// advances the period anchor-preservingly, and attempts payment with the
// customer's saved method when one exists.
//
// Ordering is deliberate: invoice FIRST (its usage_ratings claim is the
// second idempotency layer under the scheduler's lease), then the period
// advance (which makes the subscription undue), then best-effort payment
// (a decline leaves the invoice open for dunning — never fails renewal).

// renewalSubscriptionRepo is the slice of the subscription repository the
// renewal engine needs; *db.SubscriptionRepository satisfies it.
type renewalSubscriptionRepo interface {
	ClaimDueForRenewal(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error)
	Update(ctx context.Context, sub *domain.Subscription) error
}

// renewalInvoicer generates the cycle invoice; *InvoiceService satisfies it.
type renewalInvoicer interface {
	GenerateInvoice(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error)
	GenerateFinalUsageInvoice(ctx context.Context, sub *domain.Subscription, endedAt time.Time) (*domain.Invoice, error)
}

// renewalSavedCharger charges a saved payment method off-session; satisfied
// by *gateway.StripeGateway (same slice the retry worker uses). nil-safe.
type renewalSavedCharger interface {
	ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
}

// renewalPaymentLookup resolves the saved method; *db.CustomerRepository.
type renewalPaymentLookup interface {
	GetSavedPaymentMethod(ctx context.Context, customerID uuid.UUID) (stripeCustomerID, paymentMethodID string, gatewayConnectionID *uuid.UUID, err error)
}

// renewalChargerRouter picks the off-session charger for a saved card's gateway
// connection (B1 autopay). nil-safe: when unset, renewal charges every card on
// the single wired platform charger (pre-B1 behavior). Satisfied by
// *SavedCardGatewayRouter.
type renewalChargerRouter interface {
	ChargerFor(ctx context.Context, gatewayConnectionID *uuid.UUID) (SavedCardCharger, error)
}

// renewalSettler settles a paid invoice through the ledger path;
// *SubscriptionService.MarkInvoicePaid.
type renewalSettler interface {
	MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (bool, error)
}

type RenewalService struct {
	subs     renewalSubscriptionRepo
	plans    port.PlanRepository
	invoicer renewalInvoicer

	// Saved-method charging trio; all nil-safe — without them renewal
	// invoices are generated open and dunning/notifications take over.
	charger renewalSavedCharger
	lookup  renewalPaymentLookup
	settler renewalSettler
	// chargerRouter routes each charge to the gateway the card was saved on
	// (B1 autopay). nil-safe: when unset, `charger` (the platform gateway) is
	// used for every card, preserving pre-B1 behavior.
	chargerRouter renewalChargerRouter

	now func() time.Time // injectable for tests
}

func NewRenewalService(subs renewalSubscriptionRepo, plans port.PlanRepository, invoicer renewalInvoicer) *RenewalService {
	return &RenewalService{
		subs:     subs,
		plans:    plans,
		invoicer: invoicer,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// SetSavedMethodCharging wires off-session payment collection (nil-safe).
func (s *RenewalService) SetSavedMethodCharging(charger renewalSavedCharger, lookup renewalPaymentLookup, settler renewalSettler) {
	s.charger = charger
	s.lookup = lookup
	s.settler = settler
}

// SetChargerRouter wires BYO-gateway routing (B1 autopay): each saved card is
// charged on the gateway connection it was saved with. nil-safe — without it,
// every card charges on the platform `charger`.
func (s *RenewalService) SetChargerRouter(router renewalChargerRouter) {
	s.chargerRouter = router
}

// renewalClaimLease mirrors the mandate debitClaimWindow: shorter than the
// hourly worst-case tick so failures retry soon, far longer than one
// renewal takes so an in-flight subscription is never re-claimed.
const renewalClaimLease = 15 * time.Minute

// renewalBatchLimit bounds one sweep; leftovers are claimed next tick.
const renewalBatchLimit = 200

// ProcessDueRenewals claims and renews every due subscription once.
// Called by the billing-cycle scheduler; safe to call concurrently — the
// repository claim is the arbiter. Per-subscription failures are logged
// and skipped (the lease lapses and retries); a claim failure aborts the
// sweep.
func (s *RenewalService) ProcessDueRenewals(ctx context.Context) (int, error) {
	subs, err := s.subs.ClaimDueForRenewal(ctx, renewalClaimLease, renewalBatchLimit)
	if err != nil {
		return 0, fmt.Errorf("failed to claim due renewals: %w", err)
	}
	renewed := 0
	for _, sub := range subs {
		if err := s.RenewSubscription(ctx, sub); err != nil {
			slog.Error("subscription renewal failed (lease will retry)",
				"subscription_id", sub.ID, "error", err)
			continue
		}
		renewed++
	}
	return renewed, nil
}

// RenewSubscription processes one claimed subscription at period end.
//
//   - cancel_at_period_end: the elapsed period's usage is rated onto a
//     final invoice (the flat fee was already billed in advance) and the
//     subscription transitions to canceled. No new period starts.
//   - otherwise: GenerateInvoice bills next period's flat fee plus the
//     elapsed period's usage, the period advances via the anchor-preserving
//     CalculateNextBillingDate, and payment is attempted best-effort.
func (s *RenewalService) RenewSubscription(ctx context.Context, sub *domain.Subscription) error {
	now := s.now()

	if sub.CancelAtPeriodEnd {
		if _, err := s.invoicer.GenerateFinalUsageInvoice(ctx, sub, sub.CurrentPeriodEnd); err != nil {
			return fmt.Errorf("final usage invoice failed: %w", err)
		}
		sub.Status = domain.SubscriptionStatusCanceled
		sub.CanceledAt = &now
		sub.CancelAtPeriodEnd = false
		sub.UpdatedAt = now
		if err := s.subs.Update(ctx, sub); err != nil {
			return fmt.Errorf("failed to finalize period-end cancellation: %w", err)
		}
		slog.Info("subscription canceled at period end", "subscription_id", sub.ID)
		return nil
	}

	plan, err := s.plans.GetByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		return fmt.Errorf("plan unavailable for renewal: %w", err)
	}

	// Invoice for the closing period BEFORE advancing: GenerateInvoice
	// rates usage over [CurrentPeriodStart, CurrentPeriodEnd) as it stands.
	inv, err := s.invoicer.GenerateInvoice(ctx, sub)
	if err != nil {
		return fmt.Errorf("renewal invoice failed: %w", err)
	}

	// Advance the period. A subscription that was down for several cycles
	// catches up one period per tick (each elapsed window gets its own
	// invoice + usage rating rather than one merged mega-invoice).
	sub.CurrentPeriodStart = sub.CurrentPeriodEnd
	sub.CurrentPeriodEnd = sub.CalculateNextBillingDate(string(plan.IntervalUnit), plan.IntervalCount)
	sub.UpdatedAt = now
	if err := s.subs.Update(ctx, sub); err != nil {
		// The invoice exists but the period didn't advance: the lease will
		// re-claim, GenerateInvoice will run again, and the usage_ratings
		// window claim suppresses duplicate metered lines. The duplicate
		// flat-fee invoice is surfaced loudly for reconciliation.
		return fmt.Errorf("period advance failed after invoice %s (flat fee may duplicate on retry): %w", inv.ID, err)
	}

	s.attemptPayment(ctx, inv)

	slog.Info("subscription renewed",
		"subscription_id", sub.ID, "invoice_id", inv.ID,
		"total", inv.Total, "period_end", sub.CurrentPeriodEnd)
	return nil
}

// attemptPayment charges the customer's saved payment method for an open
// renewal invoice. Strictly best-effort: no saved method, a lookup error,
// or a gateway decline all leave the invoice open — dunning owns recovery.
func (s *RenewalService) attemptPayment(ctx context.Context, inv *domain.Invoice) {
	if s.charger == nil || s.lookup == nil || s.settler == nil {
		return
	}
	if inv == nil || inv.Status != domain.InvoiceStatusOpen || inv.Total <= 0 {
		return
	}
	stripeCustomerID, paymentMethodID, gatewayConnID, err := s.lookup.GetSavedPaymentMethod(ctx, inv.CustomerID)
	if err != nil || stripeCustomerID == "" || paymentMethodID == "" {
		return // nothing saved; invoice stays open
	}

	amountDue := inv.Total - inv.AmountPaid - inv.CreditApplied
	if amountDue <= 0 {
		return
	}

	// B1 autopay: charge on the gateway the card was saved on. Without a router
	// (pre-B1), every card charges on the single platform charger.
	charger := s.charger
	if s.chargerRouter != nil {
		c, rerr := s.chargerRouter.ChargerFor(ctx, gatewayConnID)
		if rerr != nil || c == nil {
			slog.Warn("renewal: could not resolve saved-card gateway; invoice left open for dunning",
				"invoice_id", inv.ID, "error", rerr)
			return
		}
		charger = c
	}

	// Idempotency key ties the charge to this invoice's renewal attempt, so
	// a re-run after a crash cannot double-charge at the gateway.
	idemKey := fmt.Sprintf("renewal-%s", inv.ID)
	result, err := charger.ChargeSavedPaymentMethod(ctx, stripeCustomerID, paymentMethodID, amountDue, inv.Currency, inv.ID.String(), idemKey)
	if err != nil || result == nil || !result.Success {
		slog.Warn("renewal payment attempt failed; invoice left open for dunning",
			"invoice_id", inv.ID, "error", err)
		return
	}
	if _, err := s.settler.MarkInvoicePaid(ctx, inv.ID); err != nil {
		slog.Error("renewal charge succeeded but settlement failed (webhook will reconcile)",
			"invoice_id", inv.ID, "error", err)
	}
}
