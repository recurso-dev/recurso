// Command demo_seed loads a rich, realistic demo dataset for a SINGLE existing
// tenant, so every dashboard use case (revenue intelligence, invoices & aging,
// dunning & recovery, credit notes, ledger, quotes, churn, GST/India, etc.) has
// data to verify against.
//
// It is deliberately UNLIKE cmd/seed, which TRUNCATEs the whole database. This
// tool:
//   - never deletes or truncates anything — it only INSERTs;
//   - is scoped to one tenant (resolved from a tenant id OR a user id);
//   - back-dates history so trend/waterfall/aging/cohort charts populate;
//   - is guarded against double-seeding (refuses if demo data already present),
//     and runs in a single transaction (all-or-nothing).
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/demo_seed --account=<tenant-or-user-uuid>
//
// Safe to point at production: it adds rows for the given tenant and touches no
// other tenant's data. Demo customers use the @demo.recurso.dev email domain and
// invoice/quote/credit-note numbers are prefixed DEMO- so they're easy to spot.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const demoDomain = "demo.recurso.dev"

var (
	flagAccount      = flag.String("account", "", "tenant id (or a user id belonging to the tenant) to seed — REQUIRED")
	flagMonths       = flag.Int("months", 15, "months of back-dated history to generate")
	flagCustomers    = flag.Int("customers", 42, "number of demo customers to create")
	flagCreateTenant = flag.Bool("create-tenant", false, "create the tenant if it does not exist (for local testing only)")
	flagDryRun       = flag.Bool("dry-run", false, "roll back instead of committing (prints the counts it would insert)")
	flagReset        = flag.Bool("reset", false, "first delete this tenant's existing demo-tagged rows, then re-seed (safe: only touches demo data)")
)

func main() {
	flag.Parse()
	if *flagAccount == "" {
		log.Fatal("--account is required (tenant id, or a user id belonging to the tenant)")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	ctx := context.Background()
	tenantID, tenantName, currency, err := resolveTenant(ctx, conn, *flagAccount)
	if err != nil {
		log.Fatalf("resolve account %q: %v", *flagAccount, err)
	}
	log.Printf("Target tenant: %s (%s) — default currency %s", tenantName, tenantID, currency)

	// Guard against double-seeding. With --reset we purge the existing demo rows
	// first (below, inside the tx); otherwise it's a hard stop.
	var n int
	_ = conn.QueryRowContext(ctx,
		`SELECT count(*) FROM customers WHERE tenant_id=$1 AND email LIKE '%@'||$2`,
		tenantID, demoDomain).Scan(&n)
	if n > 0 && !*flagReset {
		log.Fatalf("tenant already has %d demo customers — already seeded. Pass --reset to purge & re-seed.", n)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	s := &seeder{
		ctx:      ctx,
		tx:       tx,
		rng:      rand.New(rand.NewSource(42)),
		tenantID: tenantID,
		currency: currency,
		now:      time.Now().UTC(),
		months:   *flagMonths,
		reset:    *flagReset,
		counts:   map[string]int{},
	}
	s.run()

	if *flagDryRun {
		log.Println("--dry-run: rolling back (no changes committed)")
		s.report()
		return
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	committed = true
	log.Println("✅ Demo data committed.")
	s.report()
}

// resolveTenant accepts either a tenant id or a user id and returns the tenant.
func resolveTenant(ctx context.Context, conn *sql.DB, account string) (uuid.UUID, string, string, error) {
	id, err := uuid.Parse(strings.TrimSpace(account))
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("not a valid uuid: %w", err)
	}
	// Try tenants first.
	var name, currency string
	err = conn.QueryRowContext(ctx,
		`SELECT name, COALESCE(default_currency,'USD') FROM tenants WHERE id=$1`, id).Scan(&name, &currency)
	if err == nil {
		return id, name, strings.ToUpper(currency), nil
	}
	// Then treat it as a user id.
	var tenantID uuid.UUID
	err2 := conn.QueryRowContext(ctx,
		`SELECT t.id, t.name, COALESCE(t.default_currency,'USD')
		   FROM users u JOIN tenants t ON t.id=u.tenant_id WHERE u.id=$1`, id).Scan(&tenantID, &name, &currency)
	if err2 == nil {
		return tenantID, name, strings.ToUpper(currency), nil
	}
	if *flagCreateTenant {
		name = "Demo Co"
		currency = "USD"
		if _, e := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, default_currency, email, created_at, updated_at)
			 VALUES ($1,$2,'USD','admin@`+demoDomain+`', now(), now()) ON CONFLICT (id) DO NOTHING`,
			id, name); e != nil {
			return uuid.Nil, "", "", fmt.Errorf("create tenant: %w", e)
		}
		return id, name, currency, nil
	}
	return uuid.Nil, "", "", fmt.Errorf("no tenant with that id, and no user with that id (pass --create-tenant to bootstrap one for testing)")
}

type seeder struct {
	ctx      context.Context
	tx       *sql.Tx
	rng      *rand.Rand
	tenantID uuid.UUID
	currency string
	now      time.Time
	months   int
	reset    bool
	counts   map[string]int

	invoiceSeq int
	quoteSeq   int
	cnSeq      int

	arAcct, cashAcct, revAcct, taxAcct    uuid.UUID
	defAcct, recAcct, credExpAcct, ccAcct uuid.UUID
	plans                                 []plan

	// Invoice facts captured during seeding so seedEvents can emit a
	// realistic invoice/payment event stream without re-querying.
	invEvts []invEvt
}

type invEvt struct {
	id         uuid.UUID
	num        string
	sub        *subscription
	status     string
	total      int64
	amountPaid int64
	createdAt  time.Time
	paidAt     time.Time
	lastErr    string
	retryCount int
}

type plan struct {
	id         uuid.UUID
	name, code string
	interval   string // month | year
	priceByCcy map[string]int64
}

type customer struct {
	id                               uuid.UUID
	name, email, country, state, ccy string
	india                            bool
	risk                             int
	createdAt                        time.Time
}

type subscription struct {
	id           uuid.UUID
	cust         *customer
	pl           *plan
	status       string
	start        time.Time
	end          time.Time // period end or canceled_at (zero if ongoing)
	canceledAt   time.Time
	monthlyMinor int64 // normalized-to-month price in cust currency
}

func (s *seeder) exec(q string, args ...any) {
	if _, err := s.tx.ExecContext(s.ctx, q, args...); err != nil {
		log.Fatalf("exec failed: %v\n  query: %s", err, strings.TrimSpace(q)[:min(120, len(strings.TrimSpace(q)))])
	}
}

// execCount is exec + rows-affected, so the purge can report what it removed.
func (s *seeder) execCount(q string, args ...any) int64 {
	res, err := s.tx.ExecContext(s.ctx, q, args...)
	if err != nil {
		log.Fatalf("exec failed: %v\n  query: %s", err, strings.TrimSpace(q)[:min(120, len(strings.TrimSpace(q)))])
	}
	n, _ := res.RowsAffected()
	return n
}

// queryID runs an INSERT ... RETURNING id (or a SELECT) and returns the id.
func (s *seeder) queryID(q string, args ...any) uuid.UUID {
	var id uuid.UUID
	if err := s.tx.QueryRowContext(s.ctx, q, args...).Scan(&id); err != nil {
		log.Fatalf("queryID failed: %v\n  query: %s", err, strings.TrimSpace(q)[:min(120, len(strings.TrimSpace(q)))])
	}
	return id
}

func (s *seeder) bump(table string, n int) { s.counts[table] += n }

// purge deletes only this tenant's demo-tagged rows (identified by the
// @demo.recurso.dev / DEMO- markers the seeder stamps), in FK-safe order.
// Non-demo data for the tenant is never touched. Runs inside the seed tx.
func (s *seeder) purge() {
	log.Println("--reset: purging existing demo rows for this tenant… (purge v3: full FK coverage — worker, portal, churn, campaign, entity rows; in-use demo plans are kept)")
	t := s.tenantID
	demoCust := `SELECT id FROM customers WHERE tenant_id=$1 AND email LIKE '%@` + demoDomain + `'`
	// Ownership, not number pattern: demo subscriptions live-bill at renewal, so
	// the renewal worker mints REAL-numbered invoices against demo customers —
	// those are demo data too, and they FK-block the subscription purge if left.
	demoSub := `SELECT id FROM subscriptions WHERE customer_id IN (` + demoCust + `)`
	demoInv := `SELECT id FROM invoices WHERE tenant_id=$1 AND (invoice_number LIKE 'INV-DEMO-%' OR customer_id IN (` + demoCust + `) OR subscription_id IN (` + demoSub + `))`
	demoCN := `SELECT id FROM credit_notes WHERE tenant_id=$1 AND (reference LIKE 'CN-DEMO-%' OR customer_id IN (` + demoCust + `))`
	// Plans: only purge demo plans nothing else references — a non-demo
	// subscription/addon/gift on a demo plan (founder tests, gift redemptions)
	// is REAL data; deleting it would be destructive, so the plan stays.
	demoPlan := `SELECT id FROM plans WHERE tenant_id=$1 AND code LIKE 'demo\_%'
		AND id NOT IN (SELECT plan_id FROM subscriptions WHERE plan_id IS NOT NULL)
		AND id NOT IN (SELECT plan_id FROM subscription_addons WHERE plan_id IS NOT NULL)
		AND id NOT IN (SELECT plan_id FROM gifts WHERE plan_id IS NOT NULL)`
	demoCamp := `SELECT id FROM dunning_campaigns WHERE tenant_id=$1 AND name LIKE '%(demo)%'`
	demoEnt := `SELECT id FROM entities WHERE tenant_id=$1 AND invoice_prefix='EU-DEMO'`
	demoHook := `SELECT id FROM webhook_endpoints WHERE tenant_id=$1 AND url LIKE '%` + demoDomain + `%'`
	stmts := []string{
		`DELETE FROM payment_attempts WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM credit_note_applications WHERE credit_note_id IN (` + demoCN + `) OR invoice_id IN (` + demoInv + `)`,
		`DELETE FROM eu_einvoices WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM invoice_disputes WHERE invoice_id IN (` + demoInv + `) OR customer_id IN (` + demoCust + `)`,
		`DELETE FROM virtual_accounts WHERE invoice_id IN (` + demoInv + `) OR customer_id IN (` + demoCust + `)`,
		`DELETE FROM consents WHERE subscription_id IN (` + demoSub + `) OR customer_id IN (` + demoCust + `)`,
		`DELETE FROM precharge_notifications WHERE subscription_id IN (` + demoSub + `) OR customer_id IN (` + demoCust + `)`,
		`DELETE FROM progressive_billing_watermarks WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM unbilled_charges WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM ledger_transactions WHERE reference_id IN (` + demoInv + `)`,
		`DELETE FROM ledger_transactions WHERE reference_id IN (SELECT id FROM recognition_events WHERE revenue_schedule_id IN (SELECT id FROM revenue_schedules WHERE invoice_id IN (` + demoInv + `) OR subscription_id IN (` + demoSub + `)))`,
		`DELETE FROM ledger_transactions WHERE reference_id IN (` + demoCN + `)`,
		`DELETE FROM invoice_items WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM recovered_payments WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM dunning_history WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM recognition_events WHERE revenue_schedule_id IN (SELECT id FROM revenue_schedules WHERE invoice_id IN (` + demoInv + `) OR subscription_id IN (` + demoSub + `))`,
		`DELETE FROM revenue_schedules WHERE invoice_id IN (` + demoInv + `) OR subscription_id IN (` + demoSub + `)`,
		`DELETE FROM referrals WHERE referrer_id IN (` + demoCust + `) OR referred_id IN (` + demoCust + `)`,
		`DELETE FROM gifts WHERE buyer_customer_id IN (` + demoCust + `) OR redeemed_by_customer_id IN (` + demoCust + `)`,
		`DELETE FROM usage_events WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM usage_ratings WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM usage_alerts WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM wallet_transactions WHERE wallet_id IN (SELECT id FROM wallets WHERE customer_id IN (` + demoCust + `))`,
		`DELETE FROM wallets WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM plan_charges WHERE plan_id IN (` + demoPlan + `)`,
		`DELETE FROM billable_metrics WHERE tenant_id=$1 AND code IN ('api_calls','active_users')
			AND id NOT IN (SELECT metric_id FROM plan_charges WHERE metric_id IS NOT NULL)`,
		`DELETE FROM subscription_addons WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM plan_entitlements WHERE plan_id IN (` + demoPlan + `)`,
		`UPDATE subscriptions SET mandate_id=NULL WHERE mandate_id IN (SELECT id FROM mandates WHERE customer_id IN (` + demoCust + `))`,
		`DELETE FROM mandates WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM churn_feature_snapshots WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM card_expiry_notifications WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM magic_links WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM portal_sessions WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM mrr_snapshots WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM churn_alerts WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM offline_payments WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM quotes WHERE tenant_id=$1 AND quote_number LIKE 'Q-DEMO-%'`,
		`DELETE FROM credit_notes WHERE id IN (` + demoCN + `)`,
		`DELETE FROM events WHERE tenant_id=$1 AND data->>'demo'='true'`,
		`DELETE FROM invoices WHERE id IN (` + demoInv + `)`,
		`DELETE FROM subscriptions WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM dunning_campaign_executions WHERE campaign_id IN (` + demoCamp + `)`,
		`DELETE FROM dunning_campaign_steps WHERE campaign_id IN (` + demoCamp + `)`,
		`DELETE FROM dunning_campaigns WHERE id IN (` + demoCamp + `)`,
		`DELETE FROM customers WHERE tenant_id=$1 AND email LIKE '%@` + demoDomain + `'`,
		`DELETE FROM prices WHERE plan_id IN (` + demoPlan + `)`,
		`DELETE FROM plans WHERE id IN (` + demoPlan + `)`,
		`UPDATE subscriptions SET coupon_id=NULL WHERE coupon_id IN (SELECT id FROM coupons WHERE tenant_id=$1 AND code LIKE 'DEMO-%')`,
		`DELETE FROM coupons WHERE tenant_id=$1 AND code LIKE 'DEMO-%'`,
		`DELETE FROM event_deliveries WHERE webhook_endpoint_id IN (` + demoHook + `)`,
		`DELETE FROM webhook_endpoints WHERE id IN (` + demoHook + `)`,
		`DELETE FROM entity_invoice_sequences WHERE entity_id IN (` + demoEnt + `)`,
		`DELETE FROM tenant_gst_configs WHERE entity_id IN (` + demoEnt + `)`,
		`DELETE FROM tenant_irp_configs WHERE entity_id IN (` + demoEnt + `)`,
		`DELETE FROM tenant_eu_config WHERE entity_id IN (` + demoEnt + `)`,
		`DELETE FROM tenant_tax_nexus WHERE entity_id IN (` + demoEnt + `)`,
		`UPDATE ledger_accounts SET entity_id=NULL WHERE entity_id IN (` + demoEnt + `)`,
		`DELETE FROM entities WHERE id IN (` + demoEnt + `)`,
	}
	for _, q := range stmts {
		n := s.execCount(q, t)
		if n > 0 {
			// "DELETE FROM <table> ..." → table name for the log line.
			table := strings.Fields(q)[2]
			log.Printf("  · purged %-28s %d", table, n)
		}
	}
	// Demo plans still referenced by real subscriptions/addons/gifts were kept
	// on purpose — deleting them would take live data with them. Say so.
	var kept int
	if err := s.tx.QueryRowContext(s.ctx,
		`SELECT count(*) FROM plans WHERE tenant_id=$1 AND code LIKE 'demo\_%'`, t).Scan(&kept); err == nil && kept > 0 {
		log.Printf("  · kept %d in-use demo plan(s) still referenced by real subscriptions/gifts", kept)
	}
	// Tenant-level ledger accounts (Cash/Revenue/Tax/Deferred) are reused via
	// lookup-or-create on re-seed and stay in place. Per-CUSTOMER AR accounts
	// (id == customer id) must go with their purged demo customers, or every
	// reseed strands another batch of identical unlabeled "Accounts Receivable
	// (1100)" rows in the Ledger picker. Only unreferenced ones are deleted —
	// an AR account still carrying non-demo transactions is history, not junk.
	s.exec(`DELETE FROM ledger_accounts a
		WHERE a.tenant_id = $1 AND a.code = 1100
		AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = a.id)
		AND NOT EXISTS (SELECT 1 FROM ledger_transactions x
			WHERE a.id IN (x.debit_account_id, x.credit_account_id))`, t)
}

func (s *seeder) run() {
	if s.reset {
		s.purge()
	}
	// This runs ~2,000 inserts, one round-trip each. Against a remote DB (Neon)
	// that can take a few minutes, so log each phase to show it's alive.
	log.Println("Seeding (~2,000 rows; over a remote DB this can take a few minutes)…")
	step := func(name string, fn func()) {
		log.Printf("  · %s", name)
		fn()
	}
	step("ledger accounts", s.seedLedgerAccounts)
	step("plans & prices", s.seedPlans)
	step("coupons", s.seedCoupons)
	step("webhooks", s.seedWebhooks)
	var custs []*customer
	var subs []*subscription
	step("customers", func() { custs = s.seedCustomers(*flagCustomers) })
	step("subscriptions", func() { subs = s.seedSubscriptions(custs) })
	step("invoices + items + ledger + dunning (the bulk)", func() { s.seedInvoicesAndDownstream(subs) })
	step("mrr snapshots", func() { s.seedMRRSnapshots(subs) })
	step("usage events", func() { s.seedUsage(subs) })
	step("metering, wallets, commitments, alerts", func() { s.seedMetering(custs, subs) })
	step("mandates & add-ons", func() { s.seedMandatesAndAddons(subs) })
	step("quotes", func() { s.seedQuotes(custs) })
	step("credit notes", func() { s.seedStandaloneCreditNotes(custs) })
	step("churn alerts", func() { s.seedChurnAlerts(custs) })
	step("offline payments", func() { s.seedOfflinePayments(custs) })
	step("referrals", func() { s.seedReferrals(custs) })
	step("gifts", func() { s.seedGifts(custs) })
	step("events", func() { s.seedEvents(custs, subs) })
	step("collections extras (paused / write-offs / ACH attempts)", s.seedCollectionsExtras)
	step("second legal entity + book split", s.seedSecondEntity)
	step("ledger balances", s.recomputeLedgerBalances)
	log.Println("  · finalizing…")
}

// recomputeLedgerBalances derives each account's debits_posted / credits_posted
// / balance from its transactions. Normally LedgerRepository.CreateTransaction
// maintains these as rows post, but the seeder inserts ledger_transactions
// directly — so without this every account balance shows 0. Balance sign
// follows the repo (ENG-148): debit-normal (asset/expense) nets debits−credits;
// credit-normal (liability/equity/revenue) nets credits−debits.
func (s *seeder) recomputeLedgerBalances() {
	s.exec(`UPDATE ledger_accounts la SET
		debits_posted  = COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE debit_account_id = la.id), 0),
		credits_posted = COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE credit_account_id = la.id), 0),
		balance = CASE WHEN lower(la.type) IN ('1','5','asset','expense')
			THEN COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE debit_account_id = la.id),0)
			   - COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE credit_account_id = la.id),0)
			ELSE COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE credit_account_id = la.id),0)
			   - COALESCE((SELECT sum(amount) FROM ledger_transactions WHERE debit_account_id = la.id),0) END,
		updated_at = now()
		WHERE la.tenant_id = $1`, s.tenantID)
}

// ---- reference data ----

func (s *seeder) seedLedgerAccounts() {
	// Lookup-or-create (ledger_accounts has no natural unique key, so a plain
	// insert would duplicate on --force). Reuse the existing account by code.
	s.arAcct = s.ledgerAccount("Accounts Receivable", "asset", 1100)
	s.cashAcct = s.ledgerAccount("Cash", "asset", 1000)
	s.revAcct = s.ledgerAccount("Revenue", "revenue", 4000)
	s.taxAcct = s.ledgerAccount("Tax Payable", "liability", 2200)
	s.defAcct = s.ledgerAccount("Deferred Revenue", "liability", 2100)
	s.recAcct = s.ledgerAccount("Recognized Revenue", "revenue", 4100)
	s.credExpAcct = s.ledgerAccount("Credits & Adjustments", "expense", 5100)
	s.ccAcct = s.ledgerAccount("Customer Credit", "liability", 2300)
}

func (s *seeder) ledgerAccount(name, typ string, code int) uuid.UUID {
	var id uuid.UUID
	if err := s.tx.QueryRowContext(s.ctx,
		`SELECT id FROM ledger_accounts WHERE tenant_id=$1 AND code=$2 LIMIT 1`, s.tenantID, code).Scan(&id); err == nil {
		return id // already exists — reuse
	}
	id = uuid.New()
	s.exec(`INSERT INTO ledger_accounts (id, tenant_id, name, type, code, ledger_id, currency, balance, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,700,$6,0, now(), now())`,
		id, s.tenantID, name, typ, code, s.currency)
	s.bump("ledger_accounts", 1)
	return id
}

func (s *seeder) seedPlans() {
	defs := []struct {
		name, code, interval string
		usd, inr             int64
	}{
		{"Starter", "starter", "month", 2900, 249000},
		{"Growth", "growth", "month", 9900, 799000},
		{"Scale", "scale", "month", 29900, 2499000},
		{"Business (Annual)", "business_annual", "year", 990000, 8990000},
		{"Enterprise (Annual)", "enterprise_annual", "year", 2400000, 19900000},
	}
	for _, d := range defs {
		// Idempotent on (tenant_id, code): returns the existing plan id on re-run
		// so prices below always reference a real plan (no orphaned FK on --force).
		pid := s.queryID(`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,1,true, now(), now())
			ON CONFLICT (tenant_id, code) DO UPDATE SET updated_at=now()
			RETURNING id`,
			uuid.New(), s.tenantID, d.name, "demo_"+d.code, d.interval)
		for ccy, amt := range map[string]int64{"USD": d.usd, "INR": d.inr} {
			var exists int
			_ = s.tx.QueryRowContext(s.ctx, `SELECT 1 FROM prices WHERE plan_id=$1 AND currency=$2 LIMIT 1`, pid, ccy).Scan(&exists)
			if exists == 1 {
				continue // price already present (re-run)
			}
			s.exec(`INSERT INTO prices (id, plan_id, currency, amount, type, created_at)
				VALUES ($1,$2,$3,$4,'recurring', now())`,
				uuid.New(), pid, ccy, amt)
			s.bump("prices", 1)
		}
		s.plans = append(s.plans, plan{id: pid, name: d.name, code: d.code, interval: d.interval,
			priceByCcy: map[string]int64{"USD": d.usd, "INR": d.inr}})
		s.bump("plans", 1)
	}
}

func (s *seeder) seedCoupons() {
	coupons := []struct {
		code, dtype string
		val         int64
		dur         string
	}{
		{"DEMO-WELCOME20", "percent", 20, "once"},
		{"DEMO-SAVE10", "percent", 10, "forever"},
		{"DEMO-FLAT50", "amount", 5000, "once"},
		{"DEMO-BLACKFRIDAY", "percent", 30, "repeating"},
	}
	for _, c := range coupons {
		s.exec(`INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6, now(), now()) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, c.code, c.dtype, c.val, c.dur)
	}
	s.bump("coupons", len(coupons))
}

func (s *seeder) seedWebhooks() {
	hooks := []struct {
		url    string
		events []string
	}{
		{"https://" + demoDomain + "/webhooks/billing", []string{"invoice.paid", "invoice.payment_failed", "subscription.created", "subscription.canceled"}},
		{"https://" + demoDomain + "/webhooks/analytics", []string{"customer.created", "payment.recovered"}},
	}
	for _, h := range hooks {
		s.exec(`INSERT INTO webhook_endpoints (id, tenant_id, url, secret, events, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'active', now(), now()) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, h.url, "whsec_demo_"+randHex(s.rng, 16), pq.Array(h.events))
	}
	s.bump("webhook_endpoints", len(hooks))
}

// ---- customers ----

// USD-primary: the tenant's currency is USD, so most customers pay USD. India
// is kept as a small (~14%) cohort so the GST / e-invoice / UPI-mandate flows
// still have data, without INR dominating the headline numbers.
var geoPool = []struct {
	country, state, stateCode string
	india                     bool
}{
	{"US", "California", "CA", false}, {"US", "New York", "NY", false}, {"US", "Texas", "TX", false},
	{"US", "Washington", "WA", false}, {"US", "Massachusetts", "MA", false},
	{"GB", "England", "", false}, {"GB", "Scotland", "", false}, {"DE", "Bavaria", "", false},
	{"CA", "Ontario", "", false}, {"AU", "NSW", "", false}, {"SG", "", "", false}, {"NL", "", "", false},
	{"IN", "Maharashtra", "27", true}, {"IN", "Karnataka", "29", true},
}

var companyWords = []string{"Acme", "Globex", "Initech", "Umbrella", "Hooli", "Stark", "Wayne", "Wonka",
	"Cyberdyne", "Soylent", "Vandelay", "Pied Piper", "Massive Dynamic", "Gringotts", "Tyrell", "Nakatomi",
	"Aperture", "BlueSun", "Prestige", "Oscorp", "Zenith", "Northwind", "Contoso", "Fabrikam", "Sterling",
	"Lumon", "Wernham", "Dunder", "Monsters", "Krusty", "Bluth", "Sirius", "Weyland", "Abstergo", "Encom",
	"Rekall", "Omni", "Sabre", "Vehement", "Gekko", "Bishop", "Clampett", "Ewing", "Pearson"}

func (s *seeder) seedCustomers(n int) []*customer {
	out := make([]*customer, 0, n)
	for i := 0; i < n; i++ {
		g := geoPool[s.rng.Intn(len(geoPool))]
		word := companyWords[i%len(companyWords)]
		suffix := []string{"Inc", "LLC", "Labs", "Corp", "Group", "Technologies", "Studio", "Systems"}[s.rng.Intn(8)]
		name := fmt.Sprintf("%s %s", word, suffix)
		ccy := "USD"
		if g.india {
			ccy = "INR"
		}
		c := &customer{
			id:        uuid.New(),
			name:      name,
			email:     fmt.Sprintf("billing+%s%d@%s", strings.ToLower(word), i, demoDomain),
			country:   g.country,
			state:     g.state,
			ccy:       ccy,
			india:     g.india,
			risk:      s.rng.Intn(100),
			createdAt: s.backdate(s.rng.Intn(s.months)+1, s.rng.Intn(28)),
		}
		out = append(out, c)

		addr, _ := json.Marshal(map[string]string{
			"line1": fmt.Sprintf("%d Market St", 100+s.rng.Intn(900)), "city": "Metropolis",
			"state": g.state, "country": g.country,
		})
		// Use empty-string/zero defaults, NOT NULL: GetByID scans these into
		// non-nullable Go strings/ints, so a NULL makes the whole customer fail
		// to load (which would silently break geography, customer detail, etc.).
		gstin, pos, stateCode, taxType := "", "", "", "none"
		if g.india {
			gstin = fmt.Sprintf("%02d%s%04dZ%d", intFromCode(g.stateCode), "ABCDE", 1000+s.rng.Intn(8999), s.rng.Intn(9))
			pos = g.state
			stateCode = g.stateCode
			taxType = "gst"
		}
		brand, last4 := "", ""
		expM, expY := 0, 0
		if s.rng.Intn(100) < 70 {
			brand = []string{"visa", "mastercard", "amex", "rupay"}[s.rng.Intn(4)]
			last4 = fmt.Sprintf("%04d", s.rng.Intn(10000))
			expM = 1 + s.rng.Intn(12)
			expY = 2027 + s.rng.Intn(4)
		}
		s.exec(`INSERT INTO customers
			(id, tenant_id, email, name, created_at, updated_at, billing_address, line1, city, state, country,
			 gstin, place_of_supply, state_code, tax_type, risk_score, card_brand, card_last4, card_exp_month, card_exp_year, ledger_account_id)
			VALUES ($1,$2,$3,$4,$5,$5,$6,$7,'Metropolis',$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
			ON CONFLICT DO NOTHING`,
			c.id, s.tenantID, c.email, c.name, c.createdAt, addr,
			fmt.Sprintf("%d Market St", 100+s.rng.Intn(900)), g.state, g.country,
			gstin, pos, stateCode, taxType, c.risk, brand, last4, expM, expY, s.arAcct)
	}
	s.bump("customers", len(out))
	return out
}

// ---- subscriptions ----

func (s *seeder) seedSubscriptions(custs []*customer) []*subscription {
	var subs []*subscription
	// status distribution across customers
	for _, c := range custs {
		// ~15% of customers have no subscription (prospects)
		if s.rng.Intn(100) < 15 {
			continue
		}
		nSub := 1
		if s.rng.Intn(100) < 20 {
			nSub = 2 // some have a second (expansion / addon plan)
		}
		for k := 0; k < nSub; k++ {
			pl := &s.plans[s.rng.Intn(len(s.plans))]
			status := s.pickStatus()
			start := c.createdAt.AddDate(0, 0, s.rng.Intn(5))
			sub := &subscription{
				id: uuid.New(), cust: c, pl: pl, status: status, start: start,
				monthlyMinor: monthlyMinor(pl, c.ccy),
			}
			switch status {
			case "canceled":
				sub.canceledAt = s.between(start.AddDate(0, 2, 0), s.now.AddDate(0, 0, -3))
				sub.end = sub.canceledAt
			case "trialing":
				sub.start = s.backdate(0, s.rng.Intn(10)) // recent
			}
			subs = append(subs, sub)
			s.insertSubscription(sub)
		}
	}
	s.bump("subscriptions", len(subs))
	return subs
}

func (s *seeder) pickStatus() string {
	r := s.rng.Intn(100)
	switch {
	case r < 55:
		return "active"
	case r < 65:
		return "trialing"
	case r < 78:
		return "past_due"
	case r < 82:
		return "paused"
	default:
		return "canceled"
	}
}

func (s *seeder) insertSubscription(sub *subscription) {
	periodStart, periodEnd := s.currentPeriod(sub)
	var trialEnd, canceledAt any
	if sub.status == "trialing" {
		trialEnd = s.now.AddDate(0, 0, 4+s.rng.Intn(10))
	}
	if !sub.canceledAt.IsZero() {
		canceledAt = sub.canceledAt
	}
	meta, _ := json.Marshal(map[string]any{"demo": true, "source": "demo_seed"})
	s.exec(`INSERT INTO subscriptions
		(id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, trial_end,
		 billing_anchor, created_at, updated_at, canceled_at, metadata, cancel_at_period_end)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$6,$9,now(),$10,$11,false) ON CONFLICT DO NOTHING`,
		sub.id, s.tenantID, sub.cust.id, sub.pl.id, sub.status, periodStart, periodEnd, trialEnd,
		sub.start, canceledAt, meta)
}

func (s *seeder) currentPeriod(sub *subscription) (time.Time, time.Time) {
	step := 1
	unit := sub.pl.interval
	cur := sub.start
	for {
		var next time.Time
		if unit == "year" {
			next = cur.AddDate(step, 0, 0)
		} else {
			next = cur.AddDate(0, step, 0)
		}
		if next.After(s.now) || (!sub.end.IsZero() && next.After(sub.end)) {
			return cur, next
		}
		cur = next
	}
}

// ---- invoices + payments + dunning + ledger ----

func (s *seeder) seedInvoicesAndDownstream(subs []*subscription) {
	campaignID := s.seedDunningCampaign()
	for _, sub := range subs {
		if sub.status == "trialing" {
			continue // no invoices during trial
		}
		periods := s.invoicePeriods(sub)
		for i, p := range periods {
			isLast := i == len(periods)-1
			s.makeInvoice(sub, p, isLast, campaignID)
		}
	}
}

type period struct{ start, end time.Time }

func (s *seeder) invoicePeriods(sub *subscription) []period {
	var ps []period
	cur := sub.start
	limit := s.now
	if !sub.end.IsZero() && sub.end.Before(limit) {
		limit = sub.end
	}
	for cur.Before(limit) {
		var next time.Time
		if sub.pl.interval == "year" {
			next = cur.AddDate(1, 0, 0)
		} else {
			next = cur.AddDate(0, 1, 0)
		}
		ps = append(ps, period{cur, next})
		cur = next
	}
	return ps
}

func (s *seeder) makeInvoice(sub *subscription, p period, isLast bool, campaignID uuid.UUID) {
	c := sub.cust
	amount := sub.pl.priceByCcy[c.ccy]
	if sub.pl.interval == "month" {
		amount = sub.pl.priceByCcy[c.ccy]
	}
	subtotal := amount
	var cgst, sgst, igst, tax int64
	if c.india {
		tax = subtotal * 18 / 100
		if c.state == "Maharashtra" { // intra-state example → CGST+SGST
			cgst, sgst = tax/2, tax/2
		} else {
			igst = tax
		}
	}
	total := subtotal + tax

	// Determine status by recency + subscription state.
	status := "paid"
	var paidAt any
	amountPaid := total
	dueDate := p.end
	var nextRetry any
	retryCount := 0
	if isLast {
		switch sub.status {
		case "past_due":
			status = "past_due"
			amountPaid = 0
			retryCount = 1 + s.rng.Intn(3)
			nextRetry = s.now.AddDate(0, 0, 1+s.rng.Intn(3))
			// Spread overdue ages so the aging report shows real 0-30 / 30-60 / 60-90 / 90+ buckets.
			dueDate = s.now.AddDate(0, 0, -(10 + s.rng.Intn(100)))
		case "active":
			// most recent invoice is open until its due date passes
			if p.end.After(s.now) {
				status = "open"
				amountPaid = 0
				dueDate = p.end
			}
		case "canceled":
			// last invoice before cancel might be void ~30% of the time
			if s.rng.Intn(100) < 30 {
				status = "void"
				amountPaid = 0
			}
		}
	}
	var paidAtT time.Time
	if status == "paid" {
		paidAtT = s.between(p.start, p.end)
		paidAt = paidAtT
	}

	s.invoiceSeq++
	invID := uuid.New()
	invNum := fmt.Sprintf("INV-DEMO-%06d", s.invoiceSeq)
	eStatus := "NA"
	if c.india && status == "paid" {
		eStatus = "GENERATED"
	}
	// Past-due invoices carry the real recovery vocabulary: an owning engine
	// (scheduler|worker|campaign — the values the Collections page filters on)
	// and a plausible last decline code, so the worklist's Owner and Last
	// failure columns demo with substance instead of blanks.
	var managedBy, lastErr any
	if status == "past_due" {
		managedBy = []string{"worker", "worker", "campaign", "scheduler"}[s.rng.Intn(4)]
		lastErr = []string{
			"card_declined", "insufficient_funds", "expired_card",
			"do_not_honor", "authentication_required",
		}[s.rng.Intn(5)]
	}
	s.exec(`INSERT INTO invoices
		(id, tenant_id, customer_id, subscription_id, status, currency, subtotal, tax_amount, total,
		 amount_paid, due_date, paid_at, created_at, updated_at, invoice_number,
		 igst_amount, cgst_amount, sgst_amount, hsn_code, tds_amount, e_invoice_status, retry_count, next_retry_at,
		 dunning_managed_by, last_payment_error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now(),$14,$15,$16,$17,$18,0,$19,$20,$21,$22,$23)
		ON CONFLICT DO NOTHING`,
		invID, s.tenantID, c.id, sub.id, status, c.ccy, subtotal, tax, total,
		amountPaid, dueDate, paidAt, p.start, invNum,
		igst, cgst, sgst, "9983", eStatus, retryCount, nextRetry,
		managedBy, lastErr)
	s.bump("invoices", 1)
	lastErrCode := ""
	if v, ok := lastErr.(string); ok {
		lastErrCode = v
	}
	s.invEvts = append(s.invEvts, invEvt{
		id: invID, num: invNum, sub: sub, status: status, total: total,
		amountPaid: amountPaid, createdAt: p.start, paidAt: paidAtT,
		lastErr: lastErrCode, retryCount: retryCount,
	})

	// line item
	s.exec(`INSERT INTO invoice_items
		(id, invoice_id, description, hsn_code, quantity, unit_amount, amount, tax_rate, cgst_amount, sgst_amount, igst_amount, taxable_amount, created_at)
		VALUES ($1,$2,$3,$4,1,$5,$5,$6,$7,$8,$9,$5,$10) ON CONFLICT DO NOTHING`,
		uuid.New(), invID, sub.pl.name+" subscription", "9983", subtotal, gstRate(c.india), cgst, sgst, igst, p.start)
	s.bump("invoice_items", 1)

	// ledger postings + payment recovery for the relevant states
	// Code-1 (invoice raised) for every non-draft invoice, split into revenue +
	// GST legs (matches the app's RecordInvoice, ENG-159).
	s.postInvoiceLedger(invID, total, tax, p.start)
	if status == "paid" {
		// Code-3 (payment) for paid invoices, summing to amount_paid.
		s.postPaymentLedger(invID, amountPaid, p.end)
		s.bump("ledger_transactions", 1)
		// Revenue recognition: spread the invoice over its service period
		// (monthly plan → 1 month, annual → 12) so deferred revenue shows.
		s.seedRevSchedule(invID, sub, subtotal, p.start)
		// ~12% of paid USD invoices were recovered after an initial failure
		// (dunning win). Kept USD-only so the recovered-revenue headline reads
		// USD — the dunning card compares raw minor units, where INR paise would
		// always outrank USD cents.
		if !c.india && s.rng.Intn(100) < 12 {
			s.recordDunning(invID, campaignID, sub, total, true)
		}
	}
	if status == "past_due" {
		s.recordDunning(invID, campaignID, sub, total, false)
	}
}

// postInvoiceLedger records the Code-1 (invoice raised) posting at the gross
// total, plus — when there's GST — a separate reclassification posting that
// moves the tax out of Revenue into Tax Payable, matching the app's
// RecordInvoice (ENG-159). A distinct code avoids the unique (reference_id,
// code) collision; Code-1 still sums to the total for the reconciler.
// postInvoiceLedger mirrors the app's RecordInvoice for SUBSCRIPTION invoices
// (every seeded invoice bills a subscription): Code-1 posts AR → DEFERRED at
// the gross total — subscription revenue is earned over the period, not at
// issuance — and the GST reclass (Code-6) moves the tax out of Deferred into
// Tax Payable, leaving Deferred holding exactly the pre-tax value the rev-rec
// schedule will recognize. Posting to Revenue directly (the old behavior) left
// Deferred unfunded, so recognition drained it NEGATIVE — the wrong-sign
// balance the founder hit on the live demo tenant.
func (s *seeder) postInvoiceLedger(invID uuid.UUID, total, tax int64, at time.Time) {
	s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		VALUES ($1,$2,$3,$4,700,1,$5,'Invoice raised', $6) ON CONFLICT DO NOTHING`,
		uuid.New(), s.arAcct, s.defAcct, total, invID, at)
	s.bump("ledger_transactions", 1)
	if tax > 0 {
		s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
			VALUES ($1,$2,$3,$4,700,6,$5,'GST on invoice', $6) ON CONFLICT DO NOTHING`,
			uuid.New(), s.defAcct, s.taxAcct, tax, invID, at)
		s.bump("ledger_transactions", 1)
	}
}

// postPaymentLedger records the Code-3 (payment) posting: debit Cash, credit AR.
// The reconciler requires one for every paid invoice, summing to amount_paid.
func (s *seeder) postPaymentLedger(invID uuid.UUID, amount int64, at time.Time) {
	s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		VALUES ($1,$2,$3,$4,700,3,$5,'Payment received', $6) ON CONFLICT DO NOTHING`,
		uuid.New(), s.cashAcct, s.arAcct, amount, invID, at)
}

func (s *seeder) seedDunningCampaign() uuid.UUID {
	id := uuid.New()
	s.exec(`INSERT INTO dunning_campaigns (id, tenant_id, name, is_active, trigger_event, created_at, updated_at)
		VALUES ($1,$2,'Smart Recovery (demo)',true,'payment_failed', now(), now()) ON CONFLICT DO NOTHING`,
		id, s.tenantID)
	s.bump("dunning_campaigns", 1)
	return id
}

func (s *seeder) recordDunning(invID, campaignID uuid.UUID, sub *subscription, amount int64, recovered bool) {
	attempts := 1 + s.rng.Intn(3)
	for a := 0; a < attempts; a++ {
		// Must match the app's vocabulary ("success"/"failure"): GetHistoryStats
		// counts successes as `outcome = 'success'`. Using "recovered"/"failed"
		// makes every retry look failed (Success Rate stuck at 0%).
		outcome := "failure"
		if recovered && a == attempts-1 {
			outcome = "success"
		}
		s.exec(`INSERT INTO dunning_history (id, tenant_id, invoice_id, context_key, action_id, retry_interval, outcome, reward, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, invID, fmt.Sprintf("ctx_%s", sub.cust.ccy),
			fmt.Sprintf("retry_%dd", 1<<a), (1 << a), outcome, boolToReward(outcome == "success"),
			// Spread firings across hours of day — successes biased into
			// business hours — so the best-time-to-retry card has real
			// buckets to rank instead of one hot hour.
			s.now.AddDate(0, 0, -(attempts-a)).Add(time.Duration(s.timingHour(outcome == "success"))*time.Hour))
		s.bump("dunning_history", 1)
	}
	if recovered {
		s.exec(`INSERT INTO recovered_payments (id, tenant_id, invoice_id, amount, currency, attempts, strategy, campaign_id, days_to_recover, recovered_at)
			VALUES ($1,$2,$3,$4,$5,$6,'smart_dunning',$7,$8,$9) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, invID, amount, sub.cust.ccy, attempts, campaignID, attempts, s.now.AddDate(0, 0, -s.rng.Intn(20)))
		s.bump("recovered_payments", 1)
	}
}

// seedRevSchedule creates an ASC-606 revenue schedule for a paid invoice and
// its monthly recognition events — past dates recognized, future dates pending
// (so the Revenue Recognition report shows recognized + deferred balance).
func (s *seeder) seedRevSchedule(invID uuid.UUID, sub *subscription, total int64, start time.Time) {
	months := 1
	if sub.pl.interval == "year" {
		months = 12
	}
	schedID := uuid.New()
	s.exec(`INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active', now(), now()) ON CONFLICT DO NOTHING`,
		schedID, s.tenantID, invID, sub.id, total, sub.cust.ccy, start, start.AddDate(0, months, 0))
	s.bump("revenue_schedules", 1)
	per := total / int64(months)
	for i := 0; i < months; i++ {
		recDate := start.AddDate(0, i, 0)
		amt := per
		if i == months-1 {
			amt = total - per*int64(months-1) // remainder on the final period
		}
		st := "pending"
		if !recDate.After(s.now) {
			st = "recognized"
		}
		evID := uuid.New()
		s.exec(`INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6, now()) ON CONFLICT DO NOTHING`,
			evID, schedID, s.tenantID, amt, recDate, st)
		s.bump("recognition_events", 1)
		if st == "recognized" {
			// Mirror RecordRecognition: Deferred → Recognized, one Code-2 per
			// event id — (reference_id, code) uniqueness makes this idempotent
			// against the live recognition worker.
			s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
				VALUES ($1,$2,$3,$4,700,2,$5,'Revenue recognition', $6) ON CONFLICT DO NOTHING`,
				uuid.New(), s.defAcct, s.recAcct, amt, evID, recDate)
			s.bump("ledger_transactions", 1)
		}
	}
}

func (s *seeder) seedReferrals(custs []*customer) {
	if len(custs) < 6 {
		return
	}
	statuses := []string{"rewarded", "qualified", "pending", "rewarded", "qualified", "rewarded", "pending", "qualified"}
	for i := 0; i < 8; i++ {
		referrer := custs[i]
		referred := custs[len(custs)-1-i]
		if referrer.id == referred.id {
			continue
		}
		st := statuses[i%len(statuses)]
		var qualifiedAt any
		if st != "pending" {
			qualifiedAt = s.backdate(s.rng.Intn(6), s.rng.Intn(28))
		}
		reward := int64((2 + s.rng.Intn(4)) * 1000) // $20–$50
		s.exec(`INSERT INTO referrals (id, tenant_id, referrer_id, referred_id, code, status, reward_amount, currency, created_at, updated_at, qualified_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'USD', now(), now(), $8) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, referrer.id, referred.id, "REF-"+strings.ToUpper(randHex(s.rng, 6)), st, reward, qualifiedAt)
		s.bump("referrals", 1)
	}
}

func (s *seeder) seedGifts(custs []*customer) {
	if len(custs) < 2 || len(s.plans) == 0 {
		return
	}
	for i := 0; i < 6; i++ {
		buyer := custs[s.rng.Intn(len(custs))]
		pl := &s.plans[s.rng.Intn(len(s.plans))]
		dur := []int{3, 6, 12}[s.rng.Intn(3)]
		st := "purchased"
		var redeemedBy, redeemedAt any
		if s.rng.Intn(100) < 50 {
			st = "redeemed"
			redeemedBy = custs[s.rng.Intn(len(custs))].id
			redeemedAt = s.backdate(s.rng.Intn(4), s.rng.Intn(28))
		}
		s.exec(`INSERT INTO gifts (id, tenant_id, code, plan_id, buyer_customer_id, recipient_email, status, redeemed_by_customer_id, redeemed_at, duration_months, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now(), now()) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, "GIFT-"+strings.ToUpper(randHex(s.rng, 6)), pl.id, buyer.id,
			fmt.Sprintf("gift.recipient%d@%s", i, demoDomain), st, redeemedBy, redeemedAt, dur)
		s.bump("gifts", 1)
	}
}

// ---- MRR snapshots (drives waterfall / trend / NDR) ----

func (s *seeder) seedMRRSnapshots(subs []*subscription) {
	n := 0
	for m := s.months; m >= 0; m-- {
		snap := monthStart(s.backdate(m, 0))
		for _, sub := range subs {
			if sub.status == "trialing" {
				continue
			}
			active := !sub.start.After(monthEnd(snap))
			if !sub.end.IsZero() && sub.end.Before(snap) {
				active = false
			}
			if !active {
				continue
			}
			s.exec(`INSERT INTO mrr_snapshots (tenant_id, subscription_id, snapshot_date, mrr_amount, currency, customer_id, plan_id, created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7, now()) ON CONFLICT DO NOTHING`,
				s.tenantID, sub.id, snap.Format("2006-01-02"), sub.monthlyMinor, sub.cust.ccy, sub.cust.id, sub.pl.id)
			n++
		}
	}
	s.bump("mrr_snapshots", n)
}

// ---- usage, mandates, addons, quotes, credit notes, churn, offline, events ----

func (s *seeder) seedUsage(subs []*subscription) {
	n := 0
	dims := []string{"api_calls", "seats", "gb_storage", "emails_sent"}
	for _, sub := range subs {
		if sub.status != "active" || s.rng.Intn(100) < 55 {
			continue
		}
		for d := 0; d < 20+s.rng.Intn(40); d++ {
			s.exec(`INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp)
				VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
				uuid.New(), sub.id, sub.cust.id, dims[s.rng.Intn(len(dims))], int64(1+s.rng.Intn(500)),
				s.backdate(0, s.rng.Intn(30)))
			n++
		}
	}
	s.bump("usage_events", n)
}

func (s *seeder) seedMandatesAndAddons(subs []*subscription) {
	nm, na := 0, 0
	for _, sub := range subs {
		if sub.cust.india && sub.status == "active" {
			s.exec(`INSERT INTO mandates (id, tenant_id, customer_id, subscription_id, mandate_type, payment_method, vpa, max_amount, frequency, status, authorized_at, activated_at, next_debit_at, created_at, updated_at)
				VALUES ($1,$2,$3,$4,'upi','upi_autopay',$5,$6,'monthly','active',$7,$7,$8, now(), now()) ON CONFLICT DO NOTHING`,
				uuid.New(), s.tenantID, sub.cust.id, sub.id,
				fmt.Sprintf("%s@okhdfcbank", strings.Split(sub.cust.email, "@")[0]),
				sub.monthlyMinor*3, sub.start, s.now.AddDate(0, 0, 5+s.rng.Intn(20)))
			nm++
		}
		if s.rng.Intn(100) < 20 {
			s.exec(`INSERT INTO subscription_addons (id, tenant_id, subscription_id, plan_id, quantity, created_at)
				VALUES ($1,$2,$3,$4,$5, now()) ON CONFLICT DO NOTHING`,
				uuid.New(), s.tenantID, sub.id, sub.pl.id, 1+s.rng.Intn(5))
			na++
		}
	}
	s.bump("mandates", nm)
	s.bump("subscription_addons", na)
}

func (s *seeder) seedQuotes(custs []*customer) {
	statuses := []string{"draft", "sent", "accepted", "declined", "sent", "accepted"}
	for i := 0; i < 8; i++ {
		c := custs[s.rng.Intn(len(custs))]
		s.quoteSeq++
		sub := int64(50000 + s.rng.Intn(500000))
		tax := sub * 18 / 100
		// unit_price must equal amount when quantity is 1 — otherwise the UI renders
		// a contradictory "1 × $0.00" subtitle against a non-zero line amount (real
		// quotes stay consistent via Quote.CalculateTotals: amount = qty × unit_price).
		li, _ := json.Marshal([]map[string]any{{"description": "Enterprise onboarding", "quantity": 1, "unit_price": sub, "amount": sub}})
		s.exec(`INSERT INTO quotes (id, tenant_id, customer_id, quote_number, status, line_items, subtotal, tax_amount, discount_amount, total, currency, valid_until, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,0,$9,$10,$11,$12,now()) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, c.id, fmt.Sprintf("Q-DEMO-%04d", s.quoteSeq),
			statuses[i%len(statuses)], li, sub, tax, sub+tax, c.ccy,
			s.now.AddDate(0, 1, 0), s.backdate(s.rng.Intn(6), 0))
	}
	s.bump("quotes", 8)
}

func (s *seeder) seedStandaloneCreditNotes(custs []*customer) {
	for i := 0; i < 6; i++ {
		c := custs[s.rng.Intn(len(custs))]
		s.cnSeq++
		amt := int64(2000 + s.rng.Intn(40000))
		// Status vocabulary follows the credit-note lifecycle (ledger-backed
		// credits): 'issued' spendable, 'used' fully drawn down.
		status, refundStatus := "issued", "none"
		bal := amt
		if s.rng.Intn(100) < 50 {
			status, refundStatus, bal = "used", "processed", 0
		}
		cnID := uuid.New()
		at := s.backdate(s.rng.Intn(8), 0)
		s.exec(`INSERT INTO credit_notes (id, tenant_id, customer_id, reference, amount, balance, currency, status, reason, type, refund_status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'adjustment',$10,$11,now()) ON CONFLICT DO NOTHING`,
			cnID, s.tenantID, c.id, fmt.Sprintf("CN-DEMO-%04d", s.cnSeq), amt, bal, c.ccy, status,
			[]string{"Goodwill credit", "Service downtime", "Billing correction", "Downgrade proration"}[s.rng.Intn(4)],
			refundStatus, at)
		// The issuance leg (Code-8) the reconciler's completeness check
		// requires — mirrors RecordAdjustmentCreditIssued.
		s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
			VALUES ($1,$2,$3,$4,700,8,$5,'Credit note issued', $6) ON CONFLICT DO NOTHING`,
			uuid.New(), s.credExpAcct, s.ccAcct, amt, cnID, at)
		s.bump("ledger_transactions", 1)
	}
	s.bump("credit_notes", 6)
}

func (s *seeder) seedChurnAlerts(custs []*customer) {
	n := 0
	for _, c := range custs {
		if c.risk > 70 {
			s.exec(`INSERT INTO churn_alerts (id, tenant_id, customer_id, previous_score, new_score, threshold, alert_type, acknowledged, created_at)
				VALUES ($1,$2,$3,$4,$5,70,'high_risk',$6,$7) ON CONFLICT DO NOTHING`,
				uuid.New(), s.tenantID, c.id, c.risk-10-s.rng.Intn(10), c.risk, s.rng.Intn(100) < 30,
				s.backdate(0, s.rng.Intn(20)))
			n++
		}
	}
	s.bump("churn_alerts", n)
}

func (s *seeder) seedOfflinePayments(custs []*customer) {
	for i := 0; i < 4; i++ {
		c := custs[s.rng.Intn(len(custs))]
		s.exec(`INSERT INTO offline_payments (id, tenant_id, customer_id, payment_type, amount, currency, reference_number, notes, recorded_by, recorded_at)
			VALUES ($1,$2,$3,'bank_transfer',$4,$5,$6,'Wire received','demo_seed',$7) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, c.id, int64(100000+s.rng.Intn(900000)), c.ccy,
			fmt.Sprintf("NEFT%08d", s.rng.Intn(99999999)), s.backdate(s.rng.Intn(5), 0))
	}
	s.bump("offline_payments", 4)
}

func (s *seeder) seedEvents(custs []*customer, subs []*subscription) {
	n := 0
	// Every payload keeps "demo": true — reset-mode cleanup deletes demo
	// events by data->>'demo'='true'. The rest of the payload mirrors what
	// the live webhook pipeline would carry: a snapshot of the object at
	// the moment the event fired, so the Events page (and anyone poking the
	// sandbox API) sees production-shaped data instead of an empty marker.
	emit := func(typ, objType string, objID uuid.UUID, at time.Time, payload map[string]any) {
		payload["demo"] = true
		data, _ := json.Marshal(payload)
		s.exec(`INSERT INTO events (id, tenant_id, type, object_type, object_id, data, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, typ, objType, objID, data, at)
		n++
	}
	for _, c := range custs {
		emit("customer.created", "customer", c.id, c.createdAt, map[string]any{
			"name":     c.name,
			"email":    c.email,
			"country":  c.country,
			"currency": c.ccy,
		})
	}
	for _, sub := range subs {
		emit("subscription.created", "subscription", sub.id, sub.start, map[string]any{
			"customer_id":   sub.cust.id.String(),
			"customer_name": sub.cust.name,
			"plan":          sub.pl.name,
			"interval":      sub.pl.interval,
			"currency":      sub.cust.ccy,
			"status":        "active",
		})
		if sub.status == "canceled" && !sub.canceledAt.IsZero() {
			emit("subscription.canceled", "subscription", sub.id, sub.canceledAt, map[string]any{
				"customer_id": sub.cust.id.String(),
				"plan":        sub.pl.name,
				"canceled_at": sub.canceledAt.Format(time.RFC3339),
			})
		}
	}
	// Invoice lifecycle: created for every invoice, then the outcome —
	// paid (with a payment.succeeded companion) or payment_failed for the
	// past-due ones. This is the bulk of a real billing event stream.
	for i := range s.invEvts {
		ev := &s.invEvts[i]
		base := map[string]any{
			"invoice_number":  ev.num,
			"customer_id":     ev.sub.cust.id.String(),
			"subscription_id": ev.sub.id.String(),
			"currency":        ev.sub.cust.ccy,
			"total":           ev.total,
		}
		clone := func(extra map[string]any) map[string]any {
			out := map[string]any{}
			for k, v := range base {
				out[k] = v
			}
			for k, v := range extra {
				out[k] = v
			}
			return out
		}
		emit("invoice.created", "invoice", ev.id, ev.createdAt, clone(map[string]any{"status": "open"}))
		switch ev.status {
		case "paid":
			emit("invoice.paid", "invoice", ev.id, ev.paidAt, clone(map[string]any{
				"status":      "paid",
				"amount_paid": ev.amountPaid,
			}))
			emit("payment.succeeded", "invoice", ev.id, ev.paidAt, map[string]any{
				"invoice_number": ev.num,
				"customer_id":    ev.sub.cust.id.String(),
				"amount":         ev.amountPaid,
				"currency":       ev.sub.cust.ccy,
			})
		case "past_due":
			emit("invoice.payment_failed", "invoice", ev.id, s.between(ev.createdAt, s.now), clone(map[string]any{
				"status":      "past_due",
				"error_code":  ev.lastErr,
				"retry_count": ev.retryCount,
			}))
		}
	}
	s.bump("events", n)
}

// ---- helpers ----

func (s *seeder) backdate(monthsAgo, daysAgo int) time.Time {
	return s.now.AddDate(0, -monthsAgo, -daysAgo)
}
func (s *seeder) between(a, b time.Time) time.Time {
	if !b.After(a) {
		return a
	}
	d := b.Sub(a)
	return a.Add(time.Duration(s.rng.Int63n(int64(d))))
}

func (s *seeder) report() {
	log.Println("---- rows inserted (this tenant) ----")
	order := []string{"plans", "prices", "coupons", "webhook_endpoints", "ledger_accounts", "customers",
		"subscriptions", "invoices", "invoice_items", "ledger_transactions", "revenue_schedules",
		"recognition_events", "dunning_campaigns", "dunning_history", "recovered_payments", "mrr_snapshots",
		"usage_events", "mandates", "subscription_addons", "quotes", "credit_notes", "churn_alerts",
		"offline_payments", "referrals", "gifts", "events"}
	total := 0
	for _, k := range order {
		if v := s.counts[k]; v > 0 {
			log.Printf("  %-22s %d", k, v)
			total += v
		}
	}
	log.Printf("  %-22s %d", "TOTAL", total)
}

func monthlyMinor(p *plan, ccy string) int64 {
	amt := p.priceByCcy[ccy]
	if p.interval == "year" {
		return amt / 12
	}
	return amt
}
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
func monthEnd(t time.Time) time.Time { return monthStart(t).AddDate(0, 1, -1) }
func gstRate(india bool) string {
	if india {
		return "18.0"
	}
	return "0"
}
func boolToReward(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// timingHour picks an hour-of-day for a dunning firing: successes cluster in
// business hours (9-17), failures spread across the clock — giving the
// best-time-to-retry insight a credible signal.
func (s *seeder) timingHour(success bool) int {
	if success {
		return 9 + s.rng.Intn(9)
	}
	return s.rng.Intn(24)
}

// seedCollectionsExtras layers on the operator-facing recovery states shipped
// with Collections Intelligence: a paused invoice, manual write-offs (one
// inside the 90-day rate window, one ancient), and ACH payment attempts — one
// still settling and one returned by the bank — so the Collections page's
// chips, badges, and funnel all demo with substance.
func (s *seeder) seedCollectionsExtras() {
	// Pause one past-due invoice (operator negotiating with the customer).
	s.exec(`UPDATE invoices SET dunning_paused = TRUE
		WHERE id = (SELECT id FROM invoices WHERE tenant_id=$1 AND status='past_due'
		            AND invoice_number LIKE 'INV-DEMO-%' ORDER BY invoice_number LIMIT 1)`, s.tenantID)

	// Two write-offs: one concluded inside the 90-day cohort window, one
	// ancient — the recovery-rate denominator counts only the recent one.
	s.exec(`UPDATE invoices SET status='uncollectible', next_retry_at=NULL,
			marked_uncollectible_at = now() - interval '12 days'
		WHERE id = (SELECT id FROM invoices WHERE tenant_id=$1 AND status='past_due'
		            AND invoice_number LIKE 'INV-DEMO-%' ORDER BY invoice_number DESC LIMIT 1)`, s.tenantID)
	s.exec(`UPDATE invoices SET status='uncollectible', next_retry_at=NULL,
			marked_uncollectible_at = now() - interval '200 days'
		WHERE id = (SELECT id FROM invoices WHERE tenant_id=$1 AND status='past_due'
		            AND invoice_number LIKE 'INV-DEMO-%' ORDER BY invoice_number DESC OFFSET 1 LIMIT 1)`, s.tenantID)

	// Post the write-off ledger reversals (mirrors RecordInvoiceWriteOff):
	// DR Deferred / CR the customer's AR at pre-tax (code 22), plus the tax
	// reversal out of Tax Payable (code 23) — without these the written-off
	// invoices' deferral haunts the close pack's tie-out forever.
	s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		SELECT gen_random_uuid(), $2, $3, i.subtotal, 700, 22, i.id, 'Write-off of invoice '||i.invoice_number, i.marked_uncollectible_at
		FROM invoices i WHERE i.tenant_id=$1 AND i.status='uncollectible' AND i.invoice_number LIKE 'INV-DEMO-%'
		ON CONFLICT DO NOTHING`, s.tenantID, s.defAcct, s.arAcct)
	s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		SELECT gen_random_uuid(), $2, $3, i.tax_amount, 700, 23, i.id, 'Tax reversal on write-off of '||i.invoice_number, i.marked_uncollectible_at
		FROM invoices i WHERE i.tenant_id=$1 AND i.status='uncollectible' AND i.invoice_number LIKE 'INV-DEMO-%' AND i.tax_amount > 0
		ON CONFLICT DO NOTHING`, s.tenantID, s.taxAcct, s.arAcct)
	s.bump("ledger_transactions", 2)

	// ACH attempts: a debit still settling on an open invoice (the Collections
	// "settling" chip + dunning's in-flight guard), and a bank return (R01,
	// insufficient funds) on a past-due one (the "returned" chip).
	s.exec(`INSERT INTO payment_attempts (id, tenant_id, invoice_id, gateway, method, gateway_payment_intent_id, status, amount, created_at)
		SELECT $2, tenant_id, id, 'stripe', 'us_bank_account', 'pi_demo_settling', 'processing', total, now() - interval '1 day'
		FROM invoices WHERE tenant_id=$1 AND status='open' AND invoice_number LIKE 'INV-DEMO-%' AND currency='USD'
		ORDER BY invoice_number LIMIT 1 ON CONFLICT DO NOTHING`, s.tenantID, uuid.New())
	s.exec(`WITH target AS (
			SELECT id, tenant_id, total FROM invoices
			WHERE tenant_id=$1 AND status='past_due' AND invoice_number LIKE 'INV-DEMO-%'
			  AND currency='USD' AND NOT dunning_paused
			ORDER BY invoice_number OFFSET 1 LIMIT 1
		), att AS (
			INSERT INTO payment_attempts (id, tenant_id, invoice_id, gateway, method, gateway_payment_intent_id, status, failure_code, amount, created_at)
			SELECT $2, tenant_id, id, 'stripe', 'us_bank_account', 'pi_demo_returned', 'returned', 'R01', total, now() - interval '3 days'
			FROM target ON CONFLICT DO NOTHING
		)
		UPDATE invoices SET last_payment_error='R01' WHERE id IN (SELECT id FROM target)`, s.tenantID, uuid.New())
	s.bump("payment_attempts", 2)
}

// seedSecondEntity creates a second legal entity and moves a slice of the US
// book onto it, so the Entities control tower, MRR-by-entity, and the
// per-entity report scopes demo a real split instead of a single row.
func (s *seeder) seedSecondEntity() {
	euID := uuid.New()
	s.exec(`INSERT INTO entities (id, tenant_id, name, legal_name, is_primary, tb_ledger_id, invoice_prefix, country_code, created_at, updated_at)
		VALUES ($1,$2,'Recurso Europe','Recurso Europe B.V.',FALSE,
			(SELECT COALESCE(MAX(tb_ledger_id),1)+1 FROM entities WHERE tenant_id=$2),
			'EU-DEMO','NL', now(), now())
		ON CONFLICT DO NOTHING`, euID, s.tenantID)
	s.exec(`INSERT INTO entity_invoice_sequences (entity_id, next_number)
		SELECT id, 1 FROM entities WHERE tenant_id=$1 AND invoice_prefix='EU-DEMO'
		ON CONFLICT DO NOTHING`, s.tenantID)
	// Reassign roughly a third of the non-India book (subscriptions + their
	// invoices) to the new entity.
	s.exec(`UPDATE subscriptions SET entity_id = (SELECT id FROM entities WHERE tenant_id=$1 AND invoice_prefix='EU-DEMO')
		WHERE id IN (
			SELECT sub.id FROM subscriptions sub
			JOIN customers c ON c.id = sub.customer_id
			WHERE c.tenant_id=$1 AND c.email LIKE '%@`+demoDomain+`' AND c.country <> 'IN'
			ORDER BY sub.id LIMIT 3)`, s.tenantID)
	s.exec(`UPDATE invoices SET entity_id = (SELECT id FROM entities WHERE tenant_id=$1 AND invoice_prefix='EU-DEMO')
		WHERE subscription_id IN (
			SELECT id FROM subscriptions
			WHERE entity_id = (SELECT id FROM entities WHERE tenant_id=$1 AND invoice_prefix='EU-DEMO'))`, s.tenantID)
	s.bump("entities", 1)
}

func intFromCode(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	if n == 0 {
		n = 27
	}
	return n
}
func randHex(r *rand.Rand, n int) string {
	const hexd = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexd[r.Intn(16)]
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// seedMetering populates the v0.6.0 usage-billing surface: billable
// metrics, charges on the Growth plan, a funded wallet with movement
// history, a usage alert, and a commitment — so the metering pages,
// wallet screens, and usage-amount previews all demo well.
func (s *seeder) seedMetering(custs []*customer, subs []*subscription) {
	// Metrics (idempotent on tenant+code).
	apiCallsID := s.queryID(`INSERT INTO billable_metrics (id, tenant_id, name, code, aggregation_type, field_name, created_at, updated_at)
		VALUES ($1,$2,'API calls','api_calls','sum','', now(), now())
		ON CONFLICT (tenant_id, code) DO UPDATE SET updated_at=now() RETURNING id`,
		uuid.New(), s.tenantID)
	s.queryID(`INSERT INTO billable_metrics (id, tenant_id, name, code, aggregation_type, field_name, created_at, updated_at)
		VALUES ($1,$2,'Active users','active_users','unique','user_id', now(), now())
		ON CONFLICT (tenant_id, code) DO UPDATE SET updated_at=now() RETURNING id`,
		uuid.New(), s.tenantID)
	s.bump("billable_metrics", 2)

	// Graduated charge on the Growth plan: first 100k free, then per-call.
	var growthID uuid.UUID
	if err := s.tx.QueryRowContext(s.ctx,
		`SELECT id FROM plans WHERE tenant_id=$1 AND code='demo_growth'`, s.tenantID).Scan(&growthID); err == nil {
		amounts := `{"INR":{"tiers":[{"up_to":100000,"unit_amount":"0"},{"up_to":1000000,"unit_amount":"0.05"},{"up_to":null,"unit_amount":"0.0035"}]},` +
			`"USD":{"tiers":[{"up_to":100000,"unit_amount":"0"},{"up_to":null,"unit_amount":"0.0008"}]}}`
		s.exec(`INSERT INTO plan_charges (id, tenant_id, plan_id, metric_id, charge_model, amounts, hsn_code, created_at, updated_at)
			VALUES ($1,$2,$3,$4,'graduated',$5::jsonb,'', now(), now())
			ON CONFLICT (plan_id, metric_id) DO UPDATE SET amounts=EXCLUDED.amounts, updated_at=now()`,
			uuid.New(), s.tenantID, growthID, apiCallsID, amounts)
		s.bump("plan_charges", 1)
	}

	// A funded wallet for the first active INR customer: paid top-up,
	// expiring promotional credit, and a drain against a demo invoice.
	for _, c := range custs {
		if !c.india {
			continue
		}
		// The wallets unique key gained entity_id with Multi-Entity Books
		// (tenant, customer, entity, currency) — and entity_id is NULL for
		// primary-entity wallets, so an ON CONFLICT upsert never matches.
		// Lookup-or-create instead.
		wid := s.queryID(`WITH primary_entity AS (
				SELECT id FROM entities WHERE tenant_id=$2 AND is_primary LIMIT 1
			), ins AS (
				INSERT INTO wallets (id, tenant_id, customer_id, entity_id, currency, balance, created_at, updated_at)
				SELECT $1,$2,$3,(SELECT id FROM primary_entity),'INR',0, now(), now()
				WHERE NOT EXISTS (SELECT 1 FROM wallets
					WHERE tenant_id=$2 AND customer_id=$3 AND currency='INR'
					  AND entity_id = (SELECT id FROM primary_entity))
				RETURNING id)
			SELECT id FROM ins
			UNION ALL
			SELECT w.id FROM wallets w
			WHERE w.tenant_id=$2 AND w.customer_id=$3 AND w.currency='INR'
			  AND w.entity_id = (SELECT id FROM primary_entity)
			LIMIT 1`,
			uuid.New(), s.tenantID, c.id)
		var txCount int
		_ = s.tx.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id=$1`, wid).Scan(&txCount)
		if txCount == 0 {
			topUp := int64(50000000) // ₹5,00,000.00
			promo := int64(2500000)  // ₹25,000 promotional, expires in 20 days
			drain := int64(11800000) // one invoice paid from the wallet
			s.exec(`INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, source, amount, remaining, balance_after, created_at)
				VALUES ($1,$2,$3,'top_up','manual',$4,$5,$4,$6)`,
				uuid.New(), s.tenantID, wid, topUp, topUp-drain, s.backdate(0, 12))
			s.exec(`INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, amount, balance_after, created_at)
				VALUES ($1,$2,$3,'drain',$4,$5,$6)`,
				uuid.New(), s.tenantID, wid, drain, topUp-drain, s.backdate(0, 6))
			s.exec(`INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, source, amount, remaining, balance_after, expires_at, created_at)
				VALUES ($1,$2,$3,'top_up','promotional',$4,$4,$5, now() + interval '20 days', $6)`,
				uuid.New(), s.tenantID, wid, promo, topUp-drain+promo, s.backdate(0, 2))
			s.exec(`UPDATE wallets SET balance=$2, updated_at=now() WHERE id=$1`, wid, topUp-drain+promo)
			s.bump("wallet_transactions", 3)
		}
		s.bump("wallets", 1)
		break
	}

	// A commitment + usage alert on the first active Growth subscription.
	for _, sub := range subs {
		if sub.status != "active" || sub.pl.code != "growth" {
			continue
		}
		s.exec(`UPDATE subscriptions SET commitment_amount=$2 WHERE id=$1`, sub.id, int64(1500000)) // ₹15,000 floor
		s.exec(`INSERT INTO usage_alerts (id, tenant_id, subscription_id, metric_code, threshold_type, threshold, created_at, updated_at)
			VALUES ($1,$2,$3,'api_calls','quantity',1000000, now(), now())
			ON CONFLICT (subscription_id, metric_code, threshold_type, threshold) DO NOTHING`,
			uuid.New(), s.tenantID, sub.id)
		s.bump("usage_alerts", 1)
		break
	}
}
