package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Fakes for wallet tests (Lago-parity B1) ---

type fakeWalletRepo struct {
	port.WalletRepository
	wallet     *domain.Wallet
	created    *domain.Wallet
	topUps     []*domain.WalletTransaction
	drains     []int64 // maxAmount requested
	drainGives int64   // amount the fake "drains"
	dueList    []domain.Wallet
}

func (f *fakeWalletRepo) Create(ctx context.Context, w *domain.Wallet) error {
	f.created = w
	return nil
}

func (f *fakeWalletRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Wallet, error) {
	if f.wallet != nil && f.wallet.ID == id && f.wallet.TenantID == tenantID {
		return f.wallet, nil
	}
	return nil, nil
}

func (f *fakeWalletRepo) GetByCustomerAndCurrency(ctx context.Context, tenantID, customerID uuid.UUID, currency string) (*domain.Wallet, error) {
	if f.wallet != nil && f.wallet.CustomerID == customerID && f.wallet.Currency == currency {
		return f.wallet, nil
	}
	return nil, nil
}

func (f *fakeWalletRepo) TopUp(ctx context.Context, wtx *domain.WalletTransaction) error {
	f.topUps = append(f.topUps, wtx)
	return nil
}

func (f *fakeWalletRepo) Drain(ctx context.Context, tenantID, walletID uuid.UUID, maxAmount int64, invoiceID uuid.UUID, now time.Time) (int64, error) {
	f.drains = append(f.drains, maxAmount)
	give := f.drainGives
	if give > maxAmount {
		give = maxAmount
	}
	return give, nil
}

func (f *fakeWalletRepo) ListDueForRecharge(ctx context.Context, limit int) ([]domain.Wallet, error) {
	return f.dueList, nil
}

type fakeWalletLedger struct {
	topUps      []int64
	promos      []int64
	drainPosted []int64
}

func (f *fakeWalletLedger) RecordWalletTopUp(ctx context.Context, tenantID uuid.UUID, walletTxID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	f.topUps = append(f.topUps, amount)
	return uuid.New(), nil
}

func (f *fakeWalletLedger) RecordWalletDrain(ctx context.Context, tenantID, customerID, invoiceID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	f.drainPosted = append(f.drainPosted, amount)
	return uuid.New(), nil
}

func (f *fakeWalletLedger) RecordAdjustmentCreditIssued(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	f.promos = append(f.promos, amount)
	return uuid.New(), nil
}

type fakeWalletCustomerRepo struct {
	port.CustomerRepository
	customer *domain.Customer
}

func (f *fakeWalletCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return f.customer, nil
}

func walletFixture(balance int64) (*WalletService, *fakeWalletRepo, *fakeWalletLedger, *domain.Wallet) {
	tenantID, customerID := uuid.New(), uuid.New()
	w := &domain.Wallet{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
		Currency: "INR", Balance: balance,
	}
	repo := &fakeWalletRepo{wallet: w, drainGives: balance}
	ledger := &fakeWalletLedger{}
	custRepo := &fakeWalletCustomerRepo{customer: &domain.Customer{ID: customerID, TenantID: tenantID, Email: "c@x.com"}}
	svc := NewWalletService(repo, custRepo, ledger)
	svc.now = func() time.Time { return time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC) }
	return svc, repo, ledger, w
}

func TestWalletCreateValidation(t *testing.T) {
	svc, _, _, w := walletFixture(0)
	th := int64(1000)

	cases := []struct {
		name string
		in   CreateWalletInput
	}{
		{"bad customer id", CreateWalletInput{CustomerID: "nope", Currency: "INR"}},
		{"bad currency", CreateWalletInput{CustomerID: w.CustomerID.String(), Currency: "RUPEES"}},
		{"threshold without amount", CreateWalletInput{CustomerID: w.CustomerID.String(), Currency: "INR", AutoRechargeThreshold: &th}},
	}
	for _, tc := range cases {
		if _, err := svc.CreateWallet(context.Background(), w.TenantID, tc.in); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}

func TestWalletTopUpValidationAndLedger(t *testing.T) {
	svc, repo, ledger, w := walletFixture(0)
	future := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	past := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Rejections.
	if _, err := svc.TopUp(context.Background(), w.TenantID, w.ID, TopUpInput{Amount: 0}); err == nil {
		t.Fatal("zero amount must be rejected")
	}
	if _, err := svc.TopUp(context.Background(), w.TenantID, w.ID, TopUpInput{Amount: 100, Source: "manual", ExpiresAt: &future}); err == nil {
		t.Fatal("expiry on a paid top-up must be rejected")
	}
	if _, err := svc.TopUp(context.Background(), w.TenantID, w.ID, TopUpInput{Amount: 100, Source: "promotional", ExpiresAt: &past}); err == nil {
		t.Fatal("past expiry must be rejected")
	}

	// Paid top-up posts DR Cash / CR Customer Credit.
	if _, err := svc.TopUp(context.Background(), w.TenantID, w.ID, TopUpInput{Amount: 50000}); err != nil {
		t.Fatalf("manual top-up: %v", err)
	}
	if len(ledger.topUps) != 1 || ledger.topUps[0] != 50000 || len(ledger.promos) != 0 {
		t.Fatalf("paid top-up posted %v/%v, want cash leg only", ledger.topUps, ledger.promos)
	}

	// Promotional top-up posts the credits-issued expense leg instead.
	if _, err := svc.TopUp(context.Background(), w.TenantID, w.ID, TopUpInput{Amount: 10000, Source: "promotional", ExpiresAt: &future}); err != nil {
		t.Fatalf("promotional top-up: %v", err)
	}
	if len(ledger.promos) != 1 || ledger.promos[0] != 10000 {
		t.Fatalf("promotional top-up posted %v, want credits-issued leg", ledger.promos)
	}
	if len(repo.topUps) != 2 || repo.topUps[1].ExpiresAt == nil {
		t.Fatalf("expected 2 stored top-ups with expiry on the promo, got %+v", repo.topUps)
	}
}

func TestWalletDrainForInvoice(t *testing.T) {
	svc, repo, ledger, w := walletFixture(30000)
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: w.TenantID, CustomerID: w.CustomerID,
		Currency: "INR", Total: 118000, AmountPaid: 0, CreditApplied: 0,
		Status: domain.InvoiceStatusOpen,
	}

	drained, err := svc.DrainForInvoice(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if drained != 30000 {
		t.Fatalf("drained = %d, want the full 30000 balance", drained)
	}
	if len(repo.drains) != 1 || repo.drains[0] != 118000 {
		t.Fatalf("drain requested %v, want owed 118000", repo.drains)
	}
	if len(ledger.drainPosted) != 1 || ledger.drainPosted[0] != 30000 {
		t.Fatalf("ledger drain = %v, want [30000]", ledger.drainPosted)
	}
}

func TestWalletDrainSkipsWrongCurrencyAndEmpty(t *testing.T) {
	svc, repo, _, w := walletFixture(30000)
	usd := &domain.Invoice{ID: uuid.New(), TenantID: w.TenantID, CustomerID: w.CustomerID, Currency: "USD", Total: 1000}
	if drained, err := svc.DrainForInvoice(context.Background(), usd); err != nil || drained != 0 {
		t.Fatalf("USD invoice against INR wallet: drained %d err %v, want 0/nil", drained, err)
	}

	w.Balance = 0
	inr := &domain.Invoice{ID: uuid.New(), TenantID: w.TenantID, CustomerID: w.CustomerID, Currency: "INR", Total: 1000}
	if drained, err := svc.DrainForInvoice(context.Background(), inr); err != nil || drained != 0 {
		t.Fatalf("empty wallet: drained %d err %v, want 0/nil", drained, err)
	}
	if len(repo.drains) != 0 {
		t.Fatal("no repo drain should run for skipped invoices")
	}
}

// TestGenerateInvoice_WalletBeforeCreditNotes locks the D3 payment order:
// the wallet drains first, and the credit applier only sees what remains.
func TestGenerateInvoice_WalletBeforeCreditNotes(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"), // 18% IGST -> total 118000
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	drainer := &fakeInvoiceWalletDrainer{gives: 30000}
	svc.WalletDrainer = drainer
	applier := &fakeInvoiceCreditApplier{gives: 20000}
	svc.CreditApplier = applier

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.AmountPaid != 30000 {
		t.Fatalf("AmountPaid = %d, want the 30000 wallet drain", inv.AmountPaid)
	}
	// The credit applier's ceiling is total minus the wallet portion.
	if applier.sawCeiling != inv.Total-30000 {
		t.Fatalf("credit ceiling = %d, want %d (post-wallet remainder)", applier.sawCeiling, inv.Total-30000)
	}
	if inv.CreditApplied != 20000 {
		t.Fatalf("CreditApplied = %d, want 20000", inv.CreditApplied)
	}
	if inv.AmountDue != inv.Total-30000-20000 {
		t.Fatalf("AmountDue = %d, want %d", inv.AmountDue, inv.Total-30000-20000)
	}
	if inv.Status != domain.InvoiceStatusOpen {
		t.Fatalf("status = %q, want open (not fully covered)", inv.Status)
	}
}

func TestGenerateInvoice_WalletFullyCoversMarksPaid(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"),
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})
	svc.WalletDrainer = &fakeInvoiceWalletDrainer{gives: 1_000_000} // more than enough

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != domain.InvoiceStatusPaid || inv.AmountDue != 0 || inv.AmountPaid != inv.Total {
		t.Fatalf("status/due/paid = %q/%d/%d, want paid/0/%d", inv.Status, inv.AmountDue, inv.AmountPaid, inv.Total)
	}
}

type fakeInvoiceWalletDrainer struct{ gives int64 }

func (f *fakeInvoiceWalletDrainer) DrainForInvoice(ctx context.Context, inv *domain.Invoice) (int64, error) {
	owed := inv.Total - inv.AmountPaid - inv.CreditApplied
	if f.gives < owed {
		return f.gives, nil
	}
	return owed, nil
}

type fakeInvoiceCreditApplier struct {
	gives      int64
	sawCeiling int64
}

func (f *fakeInvoiceCreditApplier) ApplyAdjustmentCredits(ctx context.Context, tenantID, customerID uuid.UUID, currency string, invoiceID uuid.UUID, invoiceTotal int64) (int64, error) {
	f.sawCeiling = invoiceTotal
	if f.gives > invoiceTotal {
		return invoiceTotal, nil
	}
	return f.gives, nil
}

func (f *fakeInvoiceCreditApplier) SumApplicableAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, currency string) (int64, error) {
	return f.gives, nil
}

func TestWalletAutoRecharge(t *testing.T) {
	svc, repo, ledger, w := walletFixture(500)
	th, amt := int64(10000), int64(50000)
	w.AutoRechargeThreshold, w.AutoRechargeAmount = &th, &amt
	repo.dueList = []domain.Wallet{*w}

	charger := &fakeRenewalCharger{result: &port.PaymentResult{Success: true, PaymentID: "pay_w1"}}
	svc.SetSavedMethodCharging(charger, &fakeRenewalLookup{stripeID: "cus_1", methodID: "pm_1"})

	recharged, err := svc.ProcessAutoRecharges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recharged != 1 || charger.charges != 1 || charger.amount != 50000 {
		t.Fatalf("recharged/charges/amount = %d/%d/%d, want 1/1/50000", recharged, charger.charges, charger.amount)
	}
	if len(repo.topUps) != 1 || repo.topUps[0].Source != domain.WalletSourceAutoRecharge {
		t.Fatalf("top-ups = %+v, want one auto_recharge", repo.topUps)
	}
	if len(ledger.topUps) != 1 || ledger.topUps[0] != 50000 {
		t.Fatalf("ledger legs = %v, want the cash top-up", ledger.topUps)
	}
}

func TestWalletAutoRechargeNoSavedMethodOnlyNotifies(t *testing.T) {
	svc, repo, _, w := walletFixture(500)
	th, amt := int64(10000), int64(50000)
	w.AutoRechargeThreshold, w.AutoRechargeAmount = &th, &amt
	repo.dueList = []domain.Wallet{*w}

	charger := &fakeRenewalCharger{}
	svc.SetSavedMethodCharging(charger, &fakeRenewalLookup{}) // nothing saved

	recharged, err := svc.ProcessAutoRecharges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recharged != 0 || charger.charges != 0 || len(repo.topUps) != 0 {
		t.Fatalf("recharged/charges/topups = %d/%d/%d, want all zero", recharged, charger.charges, len(repo.topUps))
	}
}
