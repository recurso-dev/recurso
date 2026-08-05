package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// harnessGateway is a no-op PaymentGateway whose Refund always succeeds — the
// only method the harness exercises (gateway-path refunds). The rest return
// zero values; they are never called on the refund path.
type harnessGateway struct{}

var _ port.PaymentGateway = harnessGateway{}

func (harnessGateway) CreateOrder(context.Context, int64, string, string, string) (*port.PaymentOrder, error) {
	return nil, nil
}
func (harnessGateway) VerifyPayment(context.Context, string, string, string) error { return nil }
func (harnessGateway) CreateSubscription(context.Context, string, int, string, *int64, string) (string, error) {
	return "", nil
}
func (harnessGateway) RetryPayment(context.Context, string, int64, string) (*port.PaymentResult, error) {
	return nil, nil
}
func (harnessGateway) CreateMandate(context.Context, string, string, string, int64, string, string) (*port.MandateResult, error) {
	return nil, nil
}
func (harnessGateway) ExecuteMandateDebit(context.Context, port.MandateDebitRequest) (*port.PaymentResult, error) {
	return nil, nil
}
func (harnessGateway) RevokeMandate(context.Context, string, string, string) error { return nil }
func (harnessGateway) CreateVirtualAccount(context.Context, string, string, int64, string) (*port.VirtualAccountResult, error) {
	return nil, nil
}
func (harnessGateway) CancelSubscription(context.Context, string) error { return nil }
func (harnessGateway) Refund(context.Context, string, int64, string) (*port.RefundResult, error) {
	return &port.RefundResult{RefundID: "rfnd_harness", Status: "processed"}, nil
}

// TestLedgerInvariants_RandomizedBillingSequences is the invariant property
// harness planned in the rev-rec/ledger audit (archive PR #82 scope): it
// drives RANDOMIZED sequences of real billing operations — new paid
// subscriptions, mid-cycle upgrades and downgrades, one-off invoices,
// recognition runs, credit-note issuance, and cancels with unwind — through the
// real services, and after EVERY step asserts the reconciliation oracle finds
// an audit-grade ledger: no missing invoice OR credit-note legs, no unbalanced
// ledger, no abnormal account balances.
//
// This is the class of test that would have caught F1 (missing invoice legs
// on upgrade/mandate paths) and F3 (one-off recognition draining unfunded
// Deferred) before production: any future invoice-creating flow that forgets
// its ledger leg, or recognition path that over-drains, fails here on some
// seed instead of surfacing as reconciler drift in prod.
//
// Seeds are fixed for CI determinism; set LEDGER_INVARIANT_SEED to explore a
// specific seed. Failures print seed + step for exact reproduction.
func TestLedgerInvariants_RandomizedBillingSequences(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invariant harness")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB

	// 23 and 39 are regression seeds: their sequences drive a downgrade whose
	// credit exceeds both the schedule's pending AND the subscription's
	// recognized revenue (the shortfall is an unpaid upgrade-charge's
	// unscheduled deferral) — the case that once drove Recognized Revenue
	// wrong-sign (ENG-191e).
	seeds := []int64{1, 2, 3, 4, 5, 6, 7, 8, 23, 39}
	if s := os.Getenv("LEDGER_INVARIANT_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("LEDGER_INVARIANT_SEED %q: %v", s, err)
		}
		seeds = []int64{v}
	}
	const opsPerSeed = 25

	for _, seed := range seeds {
		seed := seed
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			h := newInvariantHarness(t, conn, dbx)
			rng := rand.New(rand.NewSource(seed))

			// Every sequence starts from one active, fully-posted subscription.
			h.opNewSubscription(rng)
			h.assertAuditGrade("initial subscription")

			for step := 0; step < opsPerSeed; step++ {
				name := h.randomOp(rng)
				h.assertAuditGrade(fmt.Sprintf("seed=%d step=%d op=%s", seed, step, name))
			}
			// Layer-2 assertion: after the full sequence, the month-end close-pack
			// Deferred identity must tie. A dropped forfeit/write-off/refund leg
			// leaves Deferred too high — invisible to the reconciler, caught here.
			h.assertClosePackTies(fmt.Sprintf("seed=%d final", seed))
		})
	}
}

// invariantSub is the harness's view of one subscription under test.
type invariantSub struct {
	id       uuid.UUID
	customer uuid.UUID
	onPricey bool
	active   bool
}

type invariantHarness struct {
	t        *testing.T
	conn     *sql.DB
	ctx      context.Context
	tctx     context.Context
	tenantID uuid.UUID

	cheapPlan  uuid.UUID // 100000 USD/month
	priceyPlan uuid.UUID // 200000 USD/month
	couponID   uuid.UUID // 80% off, forever — some subs are created discounted

	ledger    *LedgerService
	revrec    *RevRecService
	subSvc    *SubscriptionService
	quoteSvc  *QuoteService
	giftSvc   *GiftService
	creditSvc *CreditNoteService
	walletSvc *WalletService
	recon     *ReconciliationService
	closePack *ClosePackService

	subs []*invariantSub
	run  string
	n    int // uniqueness counter
}

func newInvariantHarness(t *testing.T, conn *sql.DB, dbx *sqlx.DB) *invariantHarness {
	t.Helper()
	ctx := context.Background()
	tenantID := seedRevRecTenant(t, conn)

	h := &invariantHarness{
		t:        t,
		conn:     conn,
		ctx:      ctx,
		tctx:     context.WithValue(ctx, domain.TenantIDKey, tenantID),
		tenantID: tenantID,
		run:      uuid.New().String()[:8],
	}

	seedPlan := func(name string, amt int64) uuid.UUID {
		id := uuid.New()
		mustExec(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			id, tenantID, name, name+"-"+h.run)
		mustExec(t, conn, `INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'USD',$3,'recurring')`,
			uuid.New(), id, amt)
		return id
	}
	h.cheapPlan = seedPlan("inv-basic", 100000)
	h.priceyPlan = seedPlan("inv-pro", 200000)

	// A tenant coupon (80% off, forever) so randomized sequences exercise the
	// coupon × proration × revrec surface: some subscriptions are created inside
	// a DISCOUNTED period (invoice, Deferred funding, and schedule all at the
	// discounted amount), and plan changes on them must prorate at discounted
	// prices (ENG-195) — a coupon-blind proration credits list price and mints
	// account credit the customer never paid, which the reconciler flags here.
	h.couponID = uuid.New()
	mustExec(t, conn, `INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, active, created_at, updated_at)
		VALUES ($1,$2,$3,'percent',80,'forever',TRUE,NOW(),NOW())`,
		h.couponID, tenantID, "INV80-"+h.run)

	h.ledger = NewLedgerService(nil, db.NewLedgerRepository(conn))
	subRepo := db.NewSubscriptionRepository(conn)
	h.revrec = NewRevRecService(db.NewRevRecRepository(conn), h.ledger, subRepo)
	// A US-seller tax resolver (customers here are US/USD → 0 tax, no provider):
	// ConvertTrialToActive calls the resolver unconditionally, and proration is
	// unaffected (US resolves to 0, same as the prior nil path).
	taxResolver := NewTaxResolver(harnessGSTConfigs{}, "United States", "")
	h.subSvc = NewSubscriptionService(subRepo, db.NewInvoiceRepository(conn), db.NewPlanRepository(conn),
		db.NewCustomerRepository(dbx), db.NewCouponRepository(conn), nil, h.ledger, nil, nil, db.NewTxManager(conn), h.revrec, taxResolver)
	h.subSvc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	// Real quote + gift services, wired to post their invoice legs exactly as
	// production does. These drive the SERVICE create-paths (not a direct
	// RecordInvoice), so a service that forgets its ledger leg — the Q1/Q2 class
	// — is caught here by the reconciler as missing_invoice_transaction.
	h.quoteSvc = NewQuoteService(db.NewQuoteRepository(conn), db.NewInvoiceRepository(conn))
	h.quoteSvc.SetLedgerPoster(h.ledger)
	h.quoteSvc.SetTxManager(db.NewTxManager(conn))
	giftInvSvc := &InvoiceService{InvoiceRepo: db.NewInvoiceRepository(conn), LedgerPoster: h.ledger}
	h.giftSvc = NewGiftService(db.NewGiftRepository(dbx), subRepo, giftInvSvc, db.NewPlanRepository(conn), nil)

	// Real credit-note service (no gateway — the harness issues standalone
	// adjustment credits, not gateway refunds). Create books
	// RecordAdjustmentCreditIssued referencing the note, so an issued credit note
	// that loses its leg is caught as missing_credit_note_transaction — the check
	// that had no op to exercise it until now.
	h.creditSvc = NewCreditNoteService(db.NewCreditNoteRepository(dbx), db.NewCustomerRepository(dbx), db.NewInvoiceRepository(conn), harnessGateway{})
	h.creditSvc.SetLedgerService(h.ledger)
	// Wire rev-rec so gateway refunds unwind the still-deferred portion, exactly
	// as production does — the full refund money path, ledger + revrec.
	h.creditSvc.SetRevRecService(h.revrec)

	// Real wallet service driving prepaid top-ups through the entity-scoped
	// ledger. Wired with the entity reader so CreateWallet resolves the tenant's
	// primary entity (the wallets.entity_id FK is NOT NULL) instead of nil. A
	// top-up posts DR Cash / CR Customer-Credit (code 11), so the wallet balance
	// must show up in the reconciler's Customer-Credit invariant (R-014) — a
	// dropped top-up leg diverges the ledger from wallets.balance and is flagged.
	h.walletSvc = NewWalletService(db.NewWalletRepository(conn), db.NewCustomerRepository(dbx), h.ledger)
	h.walletSvc.SetEntityReader(db.NewEntityRepository(conn))

	h.recon = NewReconciliationService(db.NewLedgerRepository(conn), nil)

	// Layer 2 of the safety net: the month-end close-pack Deferred identity
	// (ledger Closing == schedule deferred + awaiting-payment). Wired with revrec
	// (schedule side) and the unscheduled-deferral reader (open-invoice deferrals)
	// so the tie-out is exact, not structurally amber. Asserted once per seed.
	h.closePack = NewClosePackService(h.ledger, h.recon)
	h.closePack.SetRevRecService(h.revrec)
	h.closePack.SetUnscheduledDeferralReader(db.NewInvoiceRepository(conn))
	return h
}

// harnessGSTConfigs is a no-op GST config provider — the harness's tenant is a
// US seller, so India GST configs are never consulted (nil is correct).
type harnessGSTConfigs struct{}

func (harnessGSTConfigs) GetByTenantID(_ context.Context, _ uuid.UUID) (*domain.TenantGSTConfig, error) {
	return nil, nil
}

// randomOp picks and executes one weighted operation; returns its name.
func (h *invariantHarness) randomOp(rng *rand.Rand) string {
	// Weighted table: plan changes and recognition dominate; structural ops
	// (new sub, one-off, cancel) keep the population evolving.
	switch p := rng.Intn(100); {
	case p < 16:
		h.opNewSubscription(rng)
		return "new_subscription"
	case p < 34:
		return h.opPlanChange(rng, true)
	case p < 48:
		return h.opPlanChange(rng, false)
	case p < 62:
		h.opBackdateOneEvent(rng)
		h.opRecognize()
		return "backdate+recognize"
	case p < 72:
		h.opOneOffInvoice(rng)
		return "one_off_invoice"
	case p < 80:
		return h.opQuoteConversion(rng)
	case p < 86:
		return h.opGiftPurchase(rng)
	case p < 89:
		return h.opIssueCreditNote(rng)
	case p < 92:
		return h.opApplyCredit(rng)
	case p < 93:
		return h.opVoidCredit(rng)
	case p < 94:
		return h.opExpireCredit(rng)
	case p < 96:
		return h.opRefund(rng)
	case p < 97:
		return h.opWriteOff(rng)
	case p < 98:
		switch rng.Intn(3) {
		case 0:
			return h.opWalletTopUp(rng)
		case 1:
			return h.opWalletDrain(rng)
		default:
			return h.opWalletExpire(rng)
		}
	case p < 99:
		return h.opTrialConversion(rng)
	default:
		return h.opCancelWithUnwind(rng)
	}
}

// opApplyCredit exercises the account-credit DRAWDOWN path: it issues spendable
// adjustment credit to a customer, posts a fresh OPEN invoice's AR leg, then
// draws the credit down against that invoice through the real
// CreditNoteService.ApplyAdjustmentCredits (DR Customer-Credit / CR AR, code 7).
// When the credit fully covers the invoice the repo marks it status='paid' with
// amount_paid=0 (settled by credit, no cash) — a state the payment-leg and
// abnormal-balance checks must handle without a false discrepancy. The harness
// issued credits but never applied them, so the whole drawdown + credit-paid
// interaction was previously unexercised.
func (h *invariantHarness) opApplyCredit(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "apply_credit_skipped"
	}
	t := h.t
	s := h.subs[rng.Intn(len(h.subs))]

	// 1. Give the customer spendable adjustment credit.
	creditAmt := int64(3000 + rng.Intn(15000))
	if _, err := h.creditSvc.Create(h.tctx, h.tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: s.customer,
		Amount:     creditAmt,
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeAdjustment),
		Reason:     "harness credit for application",
	}); err != nil {
		t.Fatalf("issue credit for application (cust %s): %v", s.customer, err)
	}

	// 2. Post a fresh OPEN invoice with its AR leg (DR AR / CR Deferred).
	h.n++
	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-CA-%d", h.run, h.n)
	total := int64(2000 + rng.Intn(20000))
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',$4,$4,0,'open',$5,NOW(),NOW())`,
		invID, h.tenantID, s.customer, total, invNo)
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: s.customer,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (credit-app): %v", err)
	}

	// 3. Draw the credit down against the invoice.
	if _, err := h.creditSvc.ApplyAdjustmentCredits(h.tctx, h.tenantID, s.customer, nil, "USD", invID, total); err != nil {
		t.Fatalf("ApplyAdjustmentCredits (inv %s): %v", invID, err)
	}
	return "apply_credit"
}

// opWalletTopUp exercises the prepaid-wallet money path through the REAL
// WalletService: a fresh customer gets a wallet (resolved to the tenant's
// primary entity via the entity reader) and a top-up. The top-up posts
// DR Cash / CR Customer-Credit (code 11) and denormalizes wallets.balance in the
// same movement — so the reconciler's Customer-Credit invariant must count the
// wallet balance alongside adjustment credit notes (R-014). A dropped top-up leg
// leaves wallets.balance funded but Customer-Credit short, which the reconciler
// now flags as customer_credit_liability_mismatch. Never exercised against real
// Postgres before — the entity_id FK made a raw insert awkward; the service
// resolves it correctly.
func (h *invariantHarness) opWalletTopUp(rng *rand.Rand) string {
	t := h.t
	h.n++

	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Wallet Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("wallet-%s-%d@t.com", h.run, h.n), uuid.New())

	w, err := h.walletSvc.CreateWallet(h.tctx, h.tenantID, CreateWalletInput{
		CustomerID: customerID.String(),
		Currency:   "USD",
	})
	if err != nil {
		t.Fatalf("CreateWallet (cust %s): %v", customerID, err)
	}

	amount := int64(5000 + rng.Intn(50000))
	if _, err := h.walletSvc.TopUp(h.tctx, h.tenantID, w.ID, TopUpInput{
		Amount: amount,
		Source: domain.WalletSourceManual,
	}); err != nil {
		t.Fatalf("TopUp (wallet %s): %v", w.ID, err)
	}
	return "wallet_topup"
}

// opWalletDrain exercises the invoice-time DRAWDOWN through the REAL
// WalletService.DrainForInvoice: a customer with a funded wallet gets a fresh
// one-off invoice (DR AR / CR Revenue — no subscription, so it earns immediately
// and never touches Deferred), then the wallet drains against it. The drain posts
// DR Customer-Credit / CR AR (code 12) and decrements wallets.balance in lockstep,
// so the R-014 Customer-Credit invariant stays tied (Customer-Credit == wallet
// balance) AND the payment-leg check sees code 12 settle the now-paid invoice
// (amount_paid == Σ code {3,10,12}). The wallet is sized to fully cover the
// invoice, so the drain clears AR and the invoice is marked paid — a wallet-
// settled invoice with no cash leg. A dropped drain leg would leave Customer-
// Credit above wallets.balance (flagged) and the paid invoice short a settling
// leg (flagged).
func (h *invariantHarness) opWalletDrain(rng *rand.Rand) string {
	t := h.t
	h.n++

	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Wallet Drain Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("wdrain-%s-%d@t.com", h.run, h.n), uuid.New())

	w, err := h.walletSvc.CreateWallet(h.tctx, h.tenantID, CreateWalletInput{
		CustomerID: customerID.String(),
		Currency:   "USD",
	})
	if err != nil {
		t.Fatalf("CreateWallet (drain, cust %s): %v", customerID, err)
	}

	// Fund the wallet ABOVE the invoice total so the drain fully settles it.
	total := int64(2000 + rng.Intn(15000))
	topUp := total + int64(1000+rng.Intn(5000))
	if _, err := h.walletSvc.TopUp(h.tctx, h.tenantID, w.ID, TopUpInput{Amount: topUp, Source: domain.WalletSourceManual}); err != nil {
		t.Fatalf("TopUp (drain, wallet %s): %v", w.ID, err)
	}

	// A one-off invoice (no subscription → CR Revenue, not Deferred).
	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-WD-%d", h.run, h.n)
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',$4,$4,0,'open',$5,NOW(),NOW())`,
		invID, h.tenantID, customerID, total, invNo)
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: customerID,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (wallet drain): %v", err)
	}

	// Drain the wallet against the invoice (DR Customer-Credit / CR AR, code 12).
	drained, err := h.walletSvc.DrainForInvoice(h.tctx, inv)
	if err != nil {
		t.Fatalf("DrainForInvoice (inv %s): %v", invID, err)
	}
	if drained != total {
		t.Fatalf("DrainForInvoice drained %d, want full %d", drained, total)
	}
	// The wallet settled the invoice in full — mark it paid, with the drained
	// amount as amount_paid (code 12 is a payment-shaped leg in the reconciler).
	mustExec(t, h.conn, `UPDATE invoices SET status='paid', amount_paid=$2 WHERE id=$1`, invID, drained)
	return "wallet_drain"
}

// opWalletExpire exercises the promotional-credit EXPIRY sweep through the REAL
// WalletService.ExpireOverdueCredits. A promotional top-up posts DR
// Credits & Adjustments (expense) / CR Customer-Credit and denormalizes
// wallets.balance; when its residue expires the sweep zeroes the balance and
// posts the discharging leg (code 15, DR Customer-Credit / CR Credits &
// Adjustments) — reversing the liability so the GL doesn't leave promo credit
// standing after it lapsed. Net effect once fully expired: Customer-Credit and
// wallets.balance both return to zero, so the R-014 invariant ties. The op tops
// up with a valid future expiry (TopUp rejects past ones), backdates the residue
// to overdue, then runs the sweep. A dropped expiry leg would leave
// Customer-Credit funded while wallets.balance dropped — flagged.
func (h *invariantHarness) opWalletExpire(rng *rand.Rand) string {
	t := h.t
	h.n++

	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Wallet Expire Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("wexp-%s-%d@t.com", h.run, h.n), uuid.New())

	w, err := h.walletSvc.CreateWallet(h.tctx, h.tenantID, CreateWalletInput{
		CustomerID: customerID.String(),
		Currency:   "USD",
	})
	if err != nil {
		t.Fatalf("CreateWallet (expire, cust %s): %v", customerID, err)
	}

	// Promotional top-up with a valid future expiry (only promo may expire, and
	// the expiry must be in the future at top-up time).
	amount := int64(3000 + rng.Intn(20000))
	future := time.Now().UTC().Add(time.Hour)
	wtx, err := h.walletSvc.TopUp(h.tctx, h.tenantID, w.ID, TopUpInput{
		Amount:    amount,
		Source:    domain.WalletSourcePromotional,
		ExpiresAt: &future,
	})
	if err != nil {
		t.Fatalf("TopUp (promo, wallet %s): %v", w.ID, err)
	}

	// Backdate the residue so the sweep sees it as overdue, then run the real
	// expiry sweep (posts code 15 for the reclaimed residue).
	mustExec(t, h.conn, `UPDATE wallet_transactions SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, wtx.ID)
	n, err := h.walletSvc.ExpireOverdueCredits(h.ctx)
	if err != nil {
		t.Fatalf("ExpireOverdueCredits (wallet %s): %v", w.ID, err)
	}
	if n < 1 {
		t.Fatalf("ExpireOverdueCredits swept %d wallets, want >= 1", n)
	}
	return "wallet_expire"
}

// opTrialConversion drives the REAL SubscriptionService.ConvertTrialToActive —
// a core money flow (trial → first paid invoice) that uses ActivateTrialWithTx
// + CreateWithTx and posts RecordInvoice, never before exercised against real
// Postgres through the service. The first invoice is non-draft ('open'), so the
// reconciler requires its Code-1 leg.
func (h *invariantHarness) opTrialConversion(rng *rand.Rand) string {
	t := h.t
	h.n++
	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Trial Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("trial-%s-%d@t.com", h.run, h.n), uuid.New())

	subID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, trial_end, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'trialing', NOW() - INTERVAL '14 days', NOW(), NOW() - INTERVAL '14 days', NOW(), NOW(), NOW())`,
		subID, h.tenantID, customerID, h.cheapPlan)

	sub := &domain.Subscription{
		ID: subID, TenantID: h.tenantID, CustomerID: customerID, PlanID: h.cheapPlan,
		Status: domain.SubscriptionStatusTrialing,
	}
	if _, err := h.subSvc.ConvertTrialToActive(h.ctx, sub); err != nil {
		t.Fatalf("ConvertTrialToActive (sub %s): %v", subID, err)
	}
	// The now-active subscription joins the population (its first invoice funded
	// Deferred and created a recognition schedule).
	h.subs = append(h.subs, &invariantSub{id: subID, customer: customerID, active: true})
	return "trial_conversion"
}

// opQuoteConversion drives the REAL QuoteService.ConvertToInvoice — the path
// that (pre-#394) created an invoice with no ledger leg. The converted invoice
// is non-draft ('open'), so the reconciler requires its Code-1 leg; a service
// that forgets to post it fails the next assertAuditGrade as
// missing_invoice_transaction.
func (h *invariantHarness) opQuoteConversion(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "quote_skipped"
	}
	t := h.t
	h.n++
	s := h.subs[rng.Intn(len(h.subs))]
	quoteID := uuid.New()
	total := int64(3000 + rng.Intn(40000))
	mustExec(t, h.conn, `INSERT INTO quotes (id, tenant_id, customer_id, quote_number, status,
		line_items, subtotal, tax_amount, discount_amount, total, currency, valid_until, notes, terms, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'accepted','[]'::jsonb,$5,0,0,$5,'USD', NOW() + INTERVAL '30 days','','', NOW(), NOW())`,
		quoteID, h.tenantID, s.customer, fmt.Sprintf("Q-%s-%d", h.run, h.n), total)
	if _, err := h.quoteSvc.ConvertToInvoice(h.ctx, quoteID, h.tenantID); err != nil {
		t.Fatalf("ConvertToInvoice (quote %s): %v", quoteID, err)
	}
	return "quote_conversion"
}

// opGiftPurchase drives the REAL GiftService.PurchaseGift — the buyer-invoice
// path that (pre-#396) posted no ledger leg. The buyer invoice is non-draft
// ('open'), so the reconciler requires its Code-1 leg.
func (h *invariantHarness) opGiftPurchase(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "gift_skipped"
	}
	s := h.subs[rng.Intn(len(h.subs))]
	months := 1 + rng.Intn(6)
	// tctx: the gift service's plan lookup is tenant-scoped from context.
	if _, err := h.giftSvc.PurchaseGift(h.tctx, h.tenantID, s.customer, h.cheapPlan, "", months); err != nil {
		h.t.Fatalf("PurchaseGift (buyer %s): %v", s.customer, err)
	}
	return "gift_purchase"
}

// opIssueCreditNote issues a standalone adjustment credit note through the REAL
// CreditNoteService.Create on the API-key path (creatorRole "" bypasses
// maker-checker → issued immediately). Create books the note as an
// account-credit liability via RecordAdjustmentCreditIssued referencing the
// note (DR Credits & Adjustments / CR Customer-Credit — balanced, so it never
// trips the balance checks). The issued note then REQUIRES that leg: because
// the post is best-effort (a failure only logs), a credit-note create path
// that forgets its ledger leg fails the next assertAuditGrade as
// missing_credit_note_transaction. Until this op existed the harness created no
// credit notes, so that reconciler check could never fire — this gives it teeth.
func (h *invariantHarness) opIssueCreditNote(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "credit_note_skipped"
	}
	s := h.subs[rng.Intn(len(h.subs))]
	amount := int64(1000 + rng.Intn(20000))
	// tctx: the credit-note service reads the tenant from context (customer
	// lookup is tenant-scoped).
	if _, err := h.creditSvc.Create(h.tctx, h.tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: s.customer,
		Amount:     amount,
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeAdjustment),
		Reason:     "harness adjustment credit",
	}); err != nil {
		h.t.Fatalf("Create credit note (cust %s): %v", s.customer, err)
	}
	return "credit_note"
}

// opVoidCredit issues a spendable adjustment credit then voids it through the
// real CreditNoteService.Void, which reverses the Customer-Credit liability
// (RecordCreditVoid, DR Customer-Credit). With R-012's Customer-Credit invariant
// in place, a dropped void-reversal leg is caught — the ledger liability stays
// above the now-zero note balance. Exercises R-009 end to end.
func (h *invariantHarness) opVoidCredit(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "void_credit_skipped"
	}
	s := h.subs[rng.Intn(len(h.subs))]
	cn, err := h.creditSvc.Create(h.tctx, h.tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: s.customer,
		Amount:     int64(1000 + rng.Intn(15000)),
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeAdjustment),
		Reason:     "harness void",
	})
	if err != nil {
		h.t.Fatalf("issue credit to void (cust %s): %v", s.customer, err)
	}
	if _, err := h.creditSvc.Void(h.tctx, h.tenantID, cn.ID, uuid.Nil); err != nil {
		h.t.Fatalf("Void credit %s: %v", cn.ID, err)
	}
	return "void_credit"
}

// opExpireCredit issues a spendable adjustment credit already past its expiry,
// then runs ExpireDueCredits — which zeroes the balance and posts
// RecordCreditExpiry to reverse the Customer-Credit liability. R-012 catches a
// dropped expiry leg (ledger above the zeroed note balance). Exercises R-008.
func (h *invariantHarness) opExpireCredit(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "expire_credit_skipped"
	}
	s := h.subs[rng.Intn(len(h.subs))]
	past := time.Now().Add(-24 * time.Hour)
	if _, err := h.creditSvc.Create(h.tctx, h.tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: s.customer,
		Amount:     int64(1000 + rng.Intn(15000)),
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeAdjustment),
		Reason:     "harness expire",
		ExpiresAt:  &past,
	}); err != nil {
		h.t.Fatalf("issue credit to expire (cust %s): %v", s.customer, err)
	}
	if _, err := h.creditSvc.ExpireDueCredits(h.ctx); err != nil {
		h.t.Fatalf("ExpireDueCredits: %v", err)
	}
	return "expire_credit"
}

// opRefund exercises the gateway money-out path: it posts a fresh PAID invoice
// WITH a gateway_payment_id (so the refund takes the gateway path, not the
// manual one) and a recognition schedule, then issues a partial refund credit
// note through the real CreditNoteService.Create (type=refund). That drives the
// gateway Refund, RecordRefund (DR Refunds / CR Cash) referencing the note, and
// the rev-rec deferred unwind. The refund note is 'issued' and references the
// note, so a dropped RecordRefund leg is caught by the credit-note check as
// missing_credit_note_transaction. Previously the harness issued adjustment
// credits but never a gateway refund.
func (h *invariantHarness) opRefund(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "refund_skipped"
	}
	t := h.t
	h.n++
	s := h.subs[rng.Intn(len(h.subs))]

	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-RF-%d", h.run, h.n)
	total := int64(6000 + rng.Intn(30000))
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, gateway_payment_id, created_at, due_date)
		VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,$6,NOW(),NOW())`,
		invID, h.tenantID, s.customer, total, invNo, "pay_"+h.run+"_"+strconv.Itoa(h.n))
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: s.customer,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (refund): %v", err)
	}
	if err := h.ledger.RecordPayment(h.ctx, inv); err != nil {
		t.Fatalf("RecordPayment (refund): %v", err)
	}
	if err := h.revrec.CreateScheduleForInvoice(h.tctx, inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice (refund): %v", err)
	}

	// Partial refund (≤ half) so the over-refund guard passes and full-refund
	// clamps aren't the only case exercised.
	amount := int64(1000 + rng.Intn(int(total/2)))
	if _, err := h.creditSvc.Create(h.tctx, h.tenantID, uuid.Nil, "", domain.CreateCreditNoteRequest{
		CustomerID: s.customer,
		InvoiceID:  &invID,
		Amount:     amount,
		Currency:   "USD",
		Type:       string(domain.CreditNoteTypeRefund),
		Reason:     "harness refund",
	}); err != nil {
		t.Fatalf("Create refund (inv %s): %v", invID, err)
	}
	return "refund"
}

// opWriteOff exercises the bad-debt write-off path: it posts an OPEN, unpaid
// subscription invoice (RecordInvoice funds Deferred, no schedule since schedules
// are created on payment), flips it to `uncollectible` (mirroring
// MarkUncollectibleScoped), then writes it off (RecordInvoiceWriteOff: code-22
// deferred / code-26 bad-debt / code-23 tax legs, all CR A/R, summing to total).
// The new GetWriteOffLedgerMismatches check requires those legs sum to total; a
// dropped leg (the post is best-effort) is caught as missing_write_off_transaction
// — the hard-detection R-010 needed (the close-pack identity absorbs it).
func (h *invariantHarness) opWriteOff(rng *rand.Rand) string {
	if len(h.subs) == 0 {
		return "write_off_skipped"
	}
	t := h.t
	h.n++
	s := h.subs[rng.Intn(len(h.subs))]

	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-WO-%d", h.run, h.n)
	total := int64(4000 + rng.Intn(25000))
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,$4,'USD',$5,$5,0,'open',$6,NOW(),NOW())`,
		invID, h.tenantID, s.customer, s.id, total, invNo)
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: s.customer, SubscriptionID: &s.id,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (write-off): %v", err)
	}
	mustExec(t, h.conn, `UPDATE invoices SET status='uncollectible', marked_uncollectible_at=NOW(), updated_at=NOW() WHERE id=$1`, invID)
	if err := h.ledger.RecordInvoiceWriteOff(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoiceWriteOff (inv %s): %v", invID, err)
	}
	return "write_off"
}

// opNewSubscription seeds a customer + active mid-period subscription on the
// cheap plan with a PAID first invoice, fully posted: invoice leg, cash leg,
// and its recognition schedule — the same baseline every production
// subscription reaches after checkout. Roughly a third of subscriptions are
// created inside a DISCOUNTED period (the tenant's 80%-off coupon, mirroring
// what CreateSubscription produces: coupon attached, periods counter 1, the
// discounted-period flag set, and the invoice / Deferred funding / schedule
// all at the discounted amount) — so every later plan change on them drives
// the coupon-aware proration (ENG-195) and its revrec machinery through the
// reconciler.
func (h *invariantHarness) opNewSubscription(rng *rand.Rand) {
	t := h.t
	h.n++
	customerID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
		VALUES ($1,$2,$3,'Inv Cust','United States','individual',$4,NOW(),NOW())`,
		customerID, h.tenantID, fmt.Sprintf("inv-%s-%d@t.com", h.run, h.n), uuid.New())

	couponed := rng.Intn(3) == 0
	total := int64(100000)
	var cpID *uuid.UUID
	periods := 0
	if couponed {
		total = 20000 // 80% off the cheap plan's 100000
		cpID = &h.couponID
		periods = 1
	}

	subID := uuid.New()
	mustExec(t, h.conn, `INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor,
			coupon_id, coupon_periods_applied, coupon_applied_current_period, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW() - INTERVAL '15 days', $5, $6, $7, NOW(), NOW())`,
		subID, h.tenantID, customerID, h.cheapPlan, cpID, periods, couponed)

	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-%d", h.run, h.n)
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, subscription_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,$4,'USD',$6,$6,$6,'paid',$5,NOW(),NOW())`,
		invID, h.tenantID, customerID, subID, invNo, total)

	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: customerID, SubscriptionID: &subID,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (new sub): %v", err)
	}
	if err := h.ledger.RecordPayment(h.ctx, inv); err != nil {
		t.Fatalf("RecordPayment (new sub): %v", err)
	}
	if err := h.revrec.CreateScheduleForInvoice(h.tctx, inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice (new sub): %v", err)
	}
	h.subs = append(h.subs, &invariantSub{id: subID, customer: customerID, active: true})
}

// opPlanChange upgrades (or downgrades) a random eligible subscription
// through the real UpdateSubscription flow — proration invoice/credit, ledger
// postings, rev-rec adjustments and all.
func (h *invariantHarness) opPlanChange(rng *rand.Rand, up bool) string {
	var candidates []*invariantSub
	for _, s := range h.subs {
		if s.active && s.onPricey != up {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return "plan_change_skipped"
	}
	s := candidates[rng.Intn(len(candidates))]
	target := h.priceyPlan
	name := "upgrade"
	if !up {
		target = h.cheapPlan
		name = "downgrade"
	}
	if _, err := h.subSvc.UpdateSubscription(h.tctx, h.tenantID, s.id, target); err != nil {
		h.t.Fatalf("UpdateSubscription (%s, sub %s): %v", name, s.id, err)
	}
	s.onPricey = up
	return name
}

// opBackdateOneEvent simulates the passage of time: one pending recognition
// event (if any) becomes due.
func (h *invariantHarness) opBackdateOneEvent(rng *rand.Rand) {
	rows, err := h.conn.QueryContext(h.ctx,
		`SELECT id FROM recognition_events WHERE tenant_id = $1 AND status = 'pending' LIMIT 20`, h.tenantID)
	if err != nil {
		h.t.Fatalf("list pending events: %v", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			h.t.Fatalf("scan event id: %v", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if len(ids) == 0 {
		return
	}
	mustExec(h.t, h.conn, `UPDATE recognition_events SET recognition_date = NOW() - INTERVAL '1 hour' WHERE id = $1`,
		ids[rng.Intn(len(ids))])
}

func (h *invariantHarness) opRecognize() {
	if err := h.revrec.ProcessDueEvents(h.ctx); err != nil {
		h.t.Fatalf("ProcessDueEvents: %v", err)
	}
}

// opOneOffInvoice books a paid one-off (no subscription) invoice with its
// ledger legs and immediate recognition.
func (h *invariantHarness) opOneOffInvoice(rng *rand.Rand) {
	if len(h.subs) == 0 {
		return
	}
	t := h.t
	h.n++
	s := h.subs[rng.Intn(len(h.subs))]
	invID := uuid.New()
	invNo := fmt.Sprintf("INV-%s-OO-%d", h.run, h.n)
	total := int64(5000 + rng.Intn(50000))
	mustExec(t, h.conn, `INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
		VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,NOW(),NOW())`,
		invID, h.tenantID, s.customer, total, invNo)
	inv := &domain.Invoice{
		ID: invID, TenantID: h.tenantID, CustomerID: s.customer,
		InvoiceNumber: invNo, Total: total, Currency: "USD",
	}
	if err := h.ledger.RecordInvoice(h.ctx, inv); err != nil {
		t.Fatalf("RecordInvoice (one-off): %v", err)
	}
	if err := h.ledger.RecordPayment(h.ctx, inv); err != nil {
		t.Fatalf("RecordPayment (one-off): %v", err)
	}
	if err := h.revrec.CreateScheduleForInvoice(h.ctx, inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice (one-off): %v", err)
	}
}

// opCancelWithUnwind cancels a random active subscription immediately and
// forfeits its still-deferred revenue (breakage), like ENG-147 cancel.
func (h *invariantHarness) opCancelWithUnwind(rng *rand.Rand) string {
	var candidates []*invariantSub
	for _, s := range h.subs {
		if s.active {
			candidates = append(candidates, s)
		}
	}
	// Never cancel the last active sub — keep the population alive.
	if len(candidates) <= 1 {
		return "cancel_skipped"
	}
	s := candidates[rng.Intn(len(candidates))]
	mustExec(h.t, h.conn, `UPDATE subscriptions SET status = 'canceled', updated_at = NOW() WHERE id = $1`, s.id)
	if _, err := h.revrec.UnwindOnCancel(h.ctx, h.tenantID, s.id); err != nil {
		h.t.Fatalf("UnwindOnCancel (sub %s): %v", s.id, err)
	}
	s.active = false
	return "cancel+unwind"
}

// assertClosePackTies runs the month-end close pack for the current period and
// fails if the Deferred identity doesn't tie (ledger Closing == schedule
// deferred + awaiting-payment). This is the second safety-net layer: a dropped
// Deferred-reversal leg (forfeit on cancel, write-off, refund unwind) leaves
// Deferred too high — the reconciler's DeferredBelowScheduled only catches
// too-LOW, so only this identity catches too-high.
func (h *invariantHarness) assertClosePackTies(label string) {
	h.t.Helper()
	now := time.Now().UTC()
	pack, err := h.closePack.Generate(h.ctx, h.tenantID, int(now.Month()), now.Year())
	if err != nil {
		h.t.Fatalf("[%s] close pack generate: %v", label, err)
	}
	if !pack.Deferred.Ties {
		h.t.Fatalf("[%s] close-pack Deferred tie-out broken: UnexplainedDelta=%d (closing=%d, awaiting=%d)",
			label, pack.Deferred.UnexplainedDelta, pack.Deferred.Rollforward.Closing, pack.Deferred.AwaitingPayment)
	}
}

// assertAuditGrade runs the reconciliation oracle and fails on any
// completeness or balance discrepancy.
func (h *invariantHarness) assertAuditGrade(label string) {
	h.t.Helper()
	report, err := h.recon.Run(h.ctx, h.tenantID)
	if err != nil {
		h.t.Fatalf("[%s] reconciliation Run: %v", label, err)
	}
	for _, d := range report.Discrepancies {
		switch d.Type {
		case DiscrepancyMissingInvoiceTx, DiscrepancyInvoiceAmountMismatch,
			DiscrepancyMissingPaymentTx, DiscrepancyPaymentAmountMismatch,
			DiscrepancyMissingCreditNoteTx,
			DiscrepancyMissingCreditApplicationTx, DiscrepancyCreditApplicationAmountMismatch,
			DiscrepancyMissingWriteOffTx, DiscrepancyWriteOffAmountMismatch,
			DiscrepancyCustomerCreditMismatch,
			DiscrepancyOrphanedTransaction,
			DiscrepancyRecognizedExceedsInvoice,
			DiscrepancyLedgerUnbalanced, DiscrepancyAbnormalBalance, DiscrepancyDeferredBelowScheduled:
			h.t.Fatalf("[%s] ledger not audit-grade: %s %+v", label, d.Type, d)
		}
	}
}
