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
	// Update persists mutable plan fields (name, interval, active, hsn_code).
	// Code is immutable. Returns sql.ErrNoRows if no row matches id+tenant.
	Update(ctx context.Context, plan *domain.Plan) error
}

type InvoiceRepository interface {
	SumUnscheduledDeferral(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Create(ctx context.Context, invoice *domain.Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error)
	// ListPaginated returns one page of invoices (newest first); CountByTenant
	// gives the total for pagination metadata. The API list path uses these so a
	// large account can't return every invoice in one response.
	ListPaginated(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Invoice, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
	Update(ctx context.Context, invoice *domain.Invoice) error
	// MarkPaid atomically transitions an invoice to paid only if it is not
	// already paid, in a single conditional UPDATE. It returns true when this
	// call performed the transition (rowsAffected == 1) and false when the
	// invoice was already paid — so concurrent settlers can gate their
	// side-effects on the winner. amount_paid is set to the invoice total.
	MarkPaid(ctx context.Context, tenantID, invoiceID uuid.UUID, paidAt time.Time) (bool, error)
	// ReverseToUnpaid is the inverse of MarkPaid: it reopens a currently-paid
	// invoice (paid → past_due, paid_at → NULL) when the bank claws back a cleared
	// payment (an ACH return, Inc 3c). amount_paid is set to retainPaid — the
	// NON-cash portion (wallet/credit/TDS) that was NOT clawed back — so the
	// reopened invoice still owes only the cash that was returned. It is idempotent
	// via the `status = 'paid'` guard and returns true only when this call
	// performed the transition, so a redelivered return webhook can't reopen twice.
	ReverseToUnpaid(ctx context.Context, tenantID, invoiceID uuid.UUID, retainPaid int64) (bool, error)
	// VoidIfOpen atomically voids a still-open (unpaid) invoice; true only when
	// this call performed the transition. Paid/void/missing rows are untouched.
	VoidIfOpen(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error)
	// ListCollectionsQueue / CountCollectionsQueue back the operator-facing
	// collections worklist (Collections Intelligence Inc 1): currently-failing
	// invoices (past_due/uncollectible, balance owing) with recovery state,
	// customer, and latest ACH attempt status. Read-only.
	ListCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) ([]domain.CollectionsQueueItem, error)
	CountCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) (int, error)
	// GetCollectionsAtRisk / GetCollectionsFailureBreakdown aggregate the
	// currently-failing population for the recovery funnel + failure breakdown
	// (Collections Intelligence Inc 2). Read-only.
	GetCollectionsAtRisk(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsAtRiskRow, error)
	GetCollectionsFailureBreakdown(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsFailureRow, error)
	// GetOutstandingByEntity sums open AR per legal entity + currency for the
	// multi-entity overview. Read-only.
	GetOutstandingByEntity(ctx context.Context, tenantID uuid.UUID) ([]domain.EntityOutstandingRow, error)
	// CountUncollectibleSince counts invoices written off in a trailing window
	// (marked_uncollectible_at) — the written-off side of the windowed
	// recovery-rate cohort.
	CountUncollectibleSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int, error)
	// Manual collections controls (Collections Intelligence Inc 3): all
	// tenant-scoped and idempotent, returning whether a row changed.
	GetRetryEligibility(ctx context.Context, tenantID, invoiceID uuid.UUID) (domain.InvoiceRetryEligibility, error)
	RequeueForRetry(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error)
	SetDunningPaused(ctx context.Context, tenantID, invoiceID uuid.UUID, paused bool) (bool, error)
	MarkUncollectibleScoped(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error)
	GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error)
	// ClaimDueForRetry atomically leases up to `limit` due retry invoices for
	// the calling worker instance, advancing next_retry_at by `lease` so a
	// second instance can't claim the same rows in the same cycle (ADR-003).
	ClaimDueForRetry(ctx context.Context, lease time.Duration, limit int) ([]*domain.Invoice, error)
	UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error
	UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error
	MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error
	// SetGatewayPaymentID records the gateway-side payment identifier that
	// settled the invoice (needed later for API refunds).
	SetGatewayPaymentID(ctx context.Context, tenantID, invoiceID uuid.UUID, gatewayPaymentID string) error
	GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error)
	GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error)
	// ClaimFailedEInvoices atomically leases due failed e-invoices so exactly one
	// runner retries each — preventing duplicate government IRN submissions under
	// a multi-instance deploy (the distributed lock is a no-op without Redis).
	ClaimFailedEInvoices(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*domain.Invoice, error)
	UpdateEInvoiceStatus(ctx context.Context, tenantID, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error
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
	// GetByID loads one gift, tenant-scoped (nil when absent).
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Gift, error)
	// SetInvoiceID links the buyer's purchase invoice to the gift.
	SetInvoiceID(ctx context.Context, giftID, tenantID, invoiceID uuid.UUID) error
	// Cancel atomically transitions purchased->canceled, returning true only
	// for the caller that won — the single-cancel gate that prevents a double
	// credit and loses cleanly to a concurrent redemption.
	Cancel(ctx context.Context, giftID, tenantID uuid.UUID) (bool, error)
	// RevertRedemption returns a gift to purchased, used if the subscription
	// creation fails after the claim so the recipient can retry.
	RevertRedemption(ctx context.Context, giftID, tenantID uuid.UUID) error
}
