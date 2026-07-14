package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type PlanRepository interface {
	Create(ctx context.Context, plan *domain.Plan) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error)
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error)
}

type InvoiceRepository interface {
	Create(ctx context.Context, invoice *domain.Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error)
	Update(ctx context.Context, invoice *domain.Invoice) error
	// MarkPaid atomically transitions an invoice to paid only if it is not
	// already paid, in a single conditional UPDATE. It returns true when this
	// call performed the transition (rowsAffected == 1) and false when the
	// invoice was already paid — so concurrent settlers can gate their
	// side-effects on the winner. amount_paid is set to the invoice total.
	MarkPaid(ctx context.Context, invoiceID uuid.UUID, paidAt time.Time) (bool, error)
	GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error)
	UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error
	UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error
	MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error
	// SetGatewayPaymentID records the gateway-side payment identifier that
	// settled the invoice (needed later for API refunds).
	SetGatewayPaymentID(ctx context.Context, invoiceID uuid.UUID, gatewayPaymentID string) error
	GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error)
	GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error)
	// ClaimFailedEInvoices atomically leases due failed e-invoices so exactly one
	// runner retries each — preventing duplicate government IRN submissions under
	// a multi-instance deploy (the distributed lock is a no-op without Redis).
	ClaimFailedEInvoices(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*domain.Invoice, error)
	UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error
}

type ReferralRepository interface {
	Create(ctx context.Context, referral *domain.Referral) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Referral, error)
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Referral, error)
	GetByReferrerID(ctx context.Context, tenantID uuid.UUID, referrerID uuid.UUID) ([]*domain.Referral, error)
	GetByReferredID(ctx context.Context, tenantID uuid.UUID, referredID uuid.UUID) (*domain.Referral, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Referral, error)
	Update(ctx context.Context, referral *domain.Referral) error
}

type GiftRepository interface {
	Create(ctx context.Context, gift *domain.Gift) error
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Gift, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Gift, error)
	Update(ctx context.Context, gift *domain.Gift) error
	// MarkRedeemed atomically transitions a gift purchased->redeemed, returning
	// true only for the caller that won the transition. This is the single-redeem
	// gate: two concurrent redemptions can't both mint a subscription.
	MarkRedeemed(ctx context.Context, giftID, tenantID, redeemedBy uuid.UUID, at time.Time) (bool, error)
	// RevertRedemption returns a gift to purchased, used if the subscription
	// creation fails after the claim so the recipient can retry.
	RevertRedemption(ctx context.Context, giftID, tenantID uuid.UUID) error
}
