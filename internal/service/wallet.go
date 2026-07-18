package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// WalletService owns prepaid wallets (Lago-parity B1): creation, top-ups
// (paid or promotional, with optional expiry), the invoice-time drain, the
// expiry sweep, and auto-recharge. Every money movement posts balanced
// ledger legs; nil-safe on the ledger for test contexts.

// WalletValidationError marks invalid caller input (maps to HTTP 400).
type WalletValidationError string

func (e WalletValidationError) Error() string { return string(e) }

var (
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrWalletExists       = errors.New("a wallet for this customer and currency already exists")
	ErrWalletCustomerGone = errors.New("customer not found")
)

// walletCharger charges a saved payment method for auto-recharge; the same
// gateway slice the renewal engine uses. nil-safe.
type walletCharger interface {
	ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
}

// walletLedger is the slice of LedgerService wallets post through.
type walletLedger interface {
	RecordWalletTopUp(ctx context.Context, tenantID uuid.UUID, walletTxID uuid.UUID, amount int64, description string) (uuid.UUID, error)
	RecordWalletDrain(ctx context.Context, tenantID, customerID, invoiceID uuid.UUID, amount int64, description string) (uuid.UUID, error)
	RecordAdjustmentCreditIssued(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) (uuid.UUID, error)
}

type WalletService struct {
	wallets   port.WalletRepository
	customers port.CustomerRepository
	ledger    walletLedger
	notifier  port.Notifier // auto-recharge failure notices; nil-safe

	charger walletCharger        // nil-safe: without it auto-recharge only notifies
	lookup  renewalPaymentLookup // saved-method resolution (shared slice)

	now func() time.Time
}

func NewWalletService(wallets port.WalletRepository, customers port.CustomerRepository, ledger walletLedger) *WalletService {
	return &WalletService{
		wallets:   wallets,
		customers: customers,
		ledger:    ledger,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

// SetNotifier wires auto-recharge failure notifications (nil-safe).
func (s *WalletService) SetNotifier(n port.Notifier) { s.notifier = n }

// SetSavedMethodCharging wires gateway charging for auto-recharge (nil-safe).
func (s *WalletService) SetSavedMethodCharging(charger walletCharger, lookup renewalPaymentLookup) {
	s.charger = charger
	s.lookup = lookup
}

// CreateWalletInput creates a wallet, optionally with auto-recharge.
type CreateWalletInput struct {
	CustomerID            string `json:"customer_id" binding:"required"`
	Currency              string `json:"currency" binding:"required"`
	AutoRechargeThreshold *int64 `json:"auto_recharge_threshold"`
	AutoRechargeAmount    *int64 `json:"auto_recharge_amount"`
}

func (s *WalletService) CreateWallet(ctx context.Context, tenantID uuid.UUID, in CreateWalletInput) (*domain.Wallet, error) {
	customerID, err := uuid.Parse(in.CustomerID)
	if err != nil {
		return nil, WalletValidationError("invalid customer_id")
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if len(currency) != 3 {
		return nil, WalletValidationError("currency must be an ISO 3-letter code")
	}
	if err := validateAutoRecharge(in.AutoRechargeThreshold, in.AutoRechargeAmount); err != nil {
		return nil, err
	}
	customer, err := s.customers.GetByID(ctx, customerID)
	if err != nil || customer == nil || customer.TenantID != tenantID {
		return nil, ErrWalletCustomerGone
	}

	now := s.now()
	w := &domain.Wallet{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		CustomerID:            customerID,
		Currency:              currency,
		AutoRechargeThreshold: in.AutoRechargeThreshold,
		AutoRechargeAmount:    in.AutoRechargeAmount,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.wallets.Create(ctx, w); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrWalletExists
		}
		return nil, err
	}
	return w, nil
}

func validateAutoRecharge(threshold, amount *int64) error {
	if (threshold == nil) != (amount == nil) {
		return WalletValidationError("auto_recharge_threshold and auto_recharge_amount must be set together")
	}
	if threshold != nil && (*threshold <= 0 || *amount <= 0) {
		return WalletValidationError("auto-recharge threshold and amount must be positive")
	}
	return nil
}

func (s *WalletService) GetWallet(ctx context.Context, tenantID, id uuid.UUID) (*domain.Wallet, error) {
	w, err := s.wallets.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWalletNotFound
	}
	return w, nil
}

func (s *WalletService) ListCustomerWallets(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Wallet, error) {
	wallets, err := s.wallets.ListByCustomer(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if wallets == nil {
		wallets = []domain.Wallet{}
	}
	return wallets, nil
}

// TopUpInput adds balance to a wallet. Source "manual" records money
// already received (bank transfer / offline); "promotional" grants credit
// with no money movement and may carry an expiry.
type TopUpInput struct {
	Amount    int64      `json:"amount" binding:"required,gt=0"`
	Source    string     `json:"source"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *WalletService) TopUp(ctx context.Context, tenantID, walletID uuid.UUID, in TopUpInput) (*domain.WalletTransaction, error) {
	if in.Amount <= 0 {
		return nil, WalletValidationError("amount must be greater than zero")
	}
	source := in.Source
	if source == "" {
		source = domain.WalletSourceManual
	}
	if source != domain.WalletSourceManual && source != domain.WalletSourcePromotional {
		return nil, WalletValidationError("source must be manual or promotional")
	}
	if in.ExpiresAt != nil && source != domain.WalletSourcePromotional {
		return nil, WalletValidationError("only promotional top-ups may expire")
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return nil, WalletValidationError("expires_at must be in the future")
	}

	w, err := s.GetWallet(ctx, tenantID, walletID)
	if err != nil {
		return nil, err
	}

	wtx := &domain.WalletTransaction{
		ID:        uuid.New(),
		TenantID:  tenantID,
		WalletID:  w.ID,
		Type:      domain.WalletTxTopUp,
		Source:    source,
		Amount:    in.Amount,
		ExpiresAt: in.ExpiresAt,
		CreatedAt: s.now(),
	}
	if err := s.wallets.TopUp(ctx, wtx); err != nil {
		return nil, err
	}

	s.postTopUpLedger(ctx, tenantID, wtx, source)
	return wtx, nil
}

// postTopUpLedger books the top-up: paid credit moves cash, promotional
// credit books an expense. Best-effort — reconciliation catches misses.
func (s *WalletService) postTopUpLedger(ctx context.Context, tenantID uuid.UUID, wtx *domain.WalletTransaction, source string) {
	if s.ledger == nil {
		return
	}
	var err error
	if source == domain.WalletSourcePromotional {
		_, err = s.ledger.RecordAdjustmentCreditIssued(ctx, tenantID, wtx.ID, wtx.Amount, "Promotional wallet credit")
	} else {
		_, err = s.ledger.RecordWalletTopUp(ctx, tenantID, wtx.ID, wtx.Amount, fmt.Sprintf("Wallet top-up (%s)", source))
	}
	if err != nil {
		slog.Error("wallet top-up ledger posting failed", "wallet_tx_id", wtx.ID, "error", err)
	}
}

func (s *WalletService) ListTransactions(ctx context.Context, tenantID, walletID uuid.UUID, limit int) ([]domain.WalletTransaction, error) {
	if _, err := s.GetWallet(ctx, tenantID, walletID); err != nil {
		return nil, err
	}
	txs, err := s.wallets.ListTransactions(ctx, tenantID, walletID, limit)
	if err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []domain.WalletTransaction{}
	}
	return txs, nil
}

// UpdateAutoRecharge sets or clears the wallet's auto-recharge rule.
func (s *WalletService) UpdateAutoRecharge(ctx context.Context, tenantID, walletID uuid.UUID, threshold, amount *int64) error {
	if err := validateAutoRecharge(threshold, amount); err != nil {
		return err
	}
	err := s.wallets.UpdateAutoRecharge(ctx, tenantID, walletID, threshold, amount)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWalletNotFound
	}
	return err
}

// DrainForInvoice consumes wallet balance against a committed invoice
// (D3: wallet applies BEFORE adjustment credit notes and the gateway).
// Returns the amount drained; 0 when the customer holds no wallet in the
// invoice currency or it is empty. The ledger leg posts per drain.
func (s *WalletService) DrainForInvoice(ctx context.Context, inv *domain.Invoice) (int64, error) {
	if inv == nil || inv.Total <= 0 {
		return 0, nil
	}
	w, err := s.wallets.GetByCustomerAndCurrency(ctx, inv.TenantID, inv.CustomerID, inv.Currency)
	if err != nil || w == nil || w.Balance <= 0 {
		return 0, err
	}
	owed := inv.Total - inv.AmountPaid - inv.CreditApplied
	if owed <= 0 {
		return 0, nil
	}
	drained, err := s.wallets.Drain(ctx, inv.TenantID, w.ID, owed, inv.ID, s.now())
	if err != nil || drained == 0 {
		return 0, err
	}
	if s.ledger != nil {
		if _, lErr := s.ledger.RecordWalletDrain(ctx, inv.TenantID, inv.CustomerID, inv.ID, drained, "Wallet applied to invoice"); lErr != nil {
			slog.Error("wallet drain ledger posting failed", "invoice_id", inv.ID, "error", lErr)
		}
	}
	return drained, nil
}

// ExpireOverdueCredits writes off expired promotional residue (called from
// the billing-cycle sweep).
func (s *WalletService) ExpireOverdueCredits(ctx context.Context) (int, error) {
	return s.wallets.ExpireOverdue(ctx, s.now())
}

// autoRechargeBatchLimit bounds one sweep.
const autoRechargeBatchLimit = 100

// ProcessAutoRecharges tops up every wallet sitting below its threshold by
// charging the customer's saved payment method. No saved method or a
// decline notifies (tenant + customer per the spec default) and moves on —
// the wallet stays low and the next sweep retries.
func (s *WalletService) ProcessAutoRecharges(ctx context.Context) (int, error) {
	due, err := s.wallets.ListDueForRecharge(ctx, autoRechargeBatchLimit)
	if err != nil {
		return 0, err
	}
	recharged := 0
	for i := range due {
		w := due[i]
		if s.rechargeWallet(ctx, &w) {
			recharged++
		}
	}
	return recharged, nil
}

func (s *WalletService) rechargeWallet(ctx context.Context, w *domain.Wallet) bool {
	if s.charger == nil || s.lookup == nil || w.AutoRechargeAmount == nil {
		return false
	}
	stripeCustomerID, paymentMethodID, err := s.lookup.GetSavedPaymentMethod(ctx, w.CustomerID)
	if err != nil || stripeCustomerID == "" || paymentMethodID == "" {
		s.notifyRechargeFailure(ctx, w, "no saved payment method")
		return false
	}

	amount := *w.AutoRechargeAmount
	// Key on wallet + current balance so a crashed sweep retried within the
	// same balance state cannot double-charge, while a later legitimate
	// recharge (post-drain, different balance) gets a fresh key.
	idemKey := fmt.Sprintf("wallet-recharge-%s-%d", w.ID, w.Balance)
	result, err := s.charger.ChargeSavedPaymentMethod(ctx, stripeCustomerID, paymentMethodID, amount, w.Currency, w.ID.String(), idemKey)
	if err != nil || result == nil || !result.Success {
		slog.Warn("wallet auto-recharge charge failed", "wallet_id", w.ID, "error", err)
		s.notifyRechargeFailure(ctx, w, "payment failed")
		return false
	}

	wtx := &domain.WalletTransaction{
		ID:        uuid.New(),
		TenantID:  w.TenantID,
		WalletID:  w.ID,
		Type:      domain.WalletTxTopUp,
		Source:    domain.WalletSourceAutoRecharge,
		Amount:    amount,
		CreatedAt: s.now(),
	}
	if err := s.wallets.TopUp(ctx, wtx); err != nil {
		slog.Error("wallet auto-recharge charged but top-up record failed — reconcile manually",
			"wallet_id", w.ID, "amount", amount, "payment_id", result.PaymentID, "error", err)
		return false
	}
	s.postTopUpLedger(ctx, w.TenantID, wtx, domain.WalletSourceManual) // cash received
	slog.Info("wallet auto-recharged", "wallet_id", w.ID, "amount", amount)
	return true
}

func (s *WalletService) notifyRechargeFailure(ctx context.Context, w *domain.Wallet, reason string) {
	if s.notifier == nil {
		return
	}
	customer, err := s.customers.GetByID(ctx, w.CustomerID)
	if err != nil || customer == nil {
		return
	}
	subject := "Wallet auto-recharge failed"
	body := fmt.Sprintf("Auto-recharge for your %s wallet could not complete: %s. Please update your payment method.", w.Currency, reason)
	if err := s.notifier.SendEmail(ctx, customer.Email, subject, body); err != nil {
		slog.Warn("wallet recharge-failure notification failed", "wallet_id", w.ID, "error", err)
	}
}
