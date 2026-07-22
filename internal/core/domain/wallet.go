package domain

import (
	"time"

	"github.com/google/uuid"
)

// Prepaid wallets (Lago-parity B1, spec_lago_parity.md D1-D3).
//
// A wallet holds money-denominated prepaid balance per customer+currency
// (minor units — not abstract "credits"). Top-ups append transactions with
// a drainable residue; invoice generation drains the wallet FIRST (before
// adjustment credit notes, before the gateway), oldest-expiring residue
// first. Every movement is a WalletTransaction (append-only, with
// balance_after) and posts balanced ledger legs.

// WalletTransactionType classifies a wallet movement.
type WalletTransactionType string

const (
	// WalletTxTopUp adds balance (paid, promotional, or auto-recharge).
	WalletTxTopUp WalletTransactionType = "top_up"
	// WalletTxDrain consumes balance to settle an invoice.
	WalletTxDrain WalletTransactionType = "drain"
	// WalletTxExpiry writes off the expired residue of a dated top-up.
	WalletTxExpiry WalletTransactionType = "expiry"
)

// Wallet top-up sources.
const (
	// WalletSourceManual is an operator-recorded paid top-up (money already
	// received, e.g. via bank transfer/offline payment).
	WalletSourceManual = "manual"
	// WalletSourcePromotional is granted credit (no money received); it may
	// carry an expiry and is non-refundable.
	WalletSourcePromotional = "promotional"
	// WalletSourceAutoRecharge is a gateway charge triggered by the
	// balance-threshold rule.
	WalletSourceAutoRecharge = "auto_recharge"
)

// Wallet is a customer's prepaid balance in one currency.
type Wallet struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	// EntityID is the legal entity this wallet belongs to (Multi-Entity Books):
	// its balance is spendable only on that entity's invoices.
	EntityID   uuid.UUID `json:"entity_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Currency   string    `json:"currency"`
	// Balance is the drainable total in minor units (denormalized from the
	// open top-up residues; never negative).
	Balance int64 `json:"balance"`
	// Auto-recharge: when Balance falls below Threshold, charge the saved
	// payment method Amount. Both nil = disabled.
	AutoRechargeThreshold *int64    `json:"auto_recharge_threshold,omitempty"`
	AutoRechargeAmount    *int64    `json:"auto_recharge_amount,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// WalletTransaction is one append-only wallet movement.
type WalletTransaction struct {
	ID       uuid.UUID             `json:"id"`
	TenantID uuid.UUID             `json:"tenant_id"`
	WalletID uuid.UUID             `json:"wallet_id"`
	Type     WalletTransactionType `json:"type"`
	Source   string                `json:"source,omitempty"`
	// Amount is always positive; Type gives it direction.
	Amount int64 `json:"amount"`
	// Remaining is the undrained residue of a top_up row (nil for other
	// types). Drains consume residues oldest-expiring first.
	Remaining *int64 `json:"remaining,omitempty"`
	// BalanceAfter is the wallet balance after this movement.
	BalanceAfter int64 `json:"balance_after"`
	// InvoiceID links a drain to the invoice it settled.
	InvoiceID *uuid.UUID `json:"invoice_id,omitempty"`
	// ExpiresAt dates a top-up's residue (promotional credits).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
