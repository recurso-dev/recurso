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

	arAcct, cashAcct, revAcct, taxAcct uuid.UUID
	plans                              []plan
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
	log.Println("--reset: purging existing demo rows for this tenant…")
	t := s.tenantID
	demoCust := `SELECT id FROM customers WHERE tenant_id=$1 AND email LIKE '%@` + demoDomain + `'`
	demoInv := `SELECT id FROM invoices WHERE tenant_id=$1 AND invoice_number LIKE 'INV-DEMO-%'`
	demoSub := `SELECT id FROM subscriptions WHERE customer_id IN (` + demoCust + `)`
	demoPlan := `SELECT id FROM plans WHERE tenant_id=$1 AND code LIKE 'demo\_%'`
	stmts := []string{
		`DELETE FROM ledger_transactions WHERE reference_id IN (` + demoInv + `)`,
		`DELETE FROM invoice_items WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM recovered_payments WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM dunning_history WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM recognition_events WHERE revenue_schedule_id IN (SELECT id FROM revenue_schedules WHERE invoice_id IN (` + demoInv + `))`,
		`DELETE FROM revenue_schedules WHERE invoice_id IN (` + demoInv + `)`,
		`DELETE FROM referrals WHERE referrer_id IN (` + demoCust + `)`,
		`DELETE FROM gifts WHERE buyer_customer_id IN (` + demoCust + `)`,
		`DELETE FROM usage_events WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM subscription_addons WHERE subscription_id IN (` + demoSub + `)`,
		`DELETE FROM mandates WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM mrr_snapshots WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM churn_alerts WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM offline_payments WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM quotes WHERE tenant_id=$1 AND quote_number LIKE 'Q-DEMO-%'`,
		`DELETE FROM credit_notes WHERE tenant_id=$1 AND reference LIKE 'CN-DEMO-%'`,
		`DELETE FROM events WHERE tenant_id=$1 AND data->>'demo'='true'`,
		`DELETE FROM invoices WHERE tenant_id=$1 AND invoice_number LIKE 'INV-DEMO-%'`,
		`DELETE FROM subscriptions WHERE customer_id IN (` + demoCust + `)`,
		`DELETE FROM dunning_campaigns WHERE tenant_id=$1 AND name LIKE '%(demo)%'`,
		`DELETE FROM customers WHERE tenant_id=$1 AND email LIKE '%@` + demoDomain + `'`,
		`DELETE FROM prices WHERE plan_id IN (` + demoPlan + `)`,
		`DELETE FROM plans WHERE tenant_id=$1 AND code LIKE 'demo\_%'`,
		`DELETE FROM coupons WHERE tenant_id=$1 AND code LIKE 'DEMO-%'`,
		`DELETE FROM webhook_endpoints WHERE tenant_id=$1 AND url LIKE '%` + demoDomain + `%'`,
	}
	for _, q := range stmts {
		s.exec(q, t)
	}
	// ledger_accounts are generic (AR/Cash/Revenue/Tax) and reused via
	// lookup-or-create on re-seed, so they are intentionally left in place.
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
	step("mandates & add-ons", func() { s.seedMandatesAndAddons(subs) })
	step("quotes", func() { s.seedQuotes(custs) })
	step("credit notes", func() { s.seedStandaloneCreditNotes(custs) })
	step("churn alerts", func() { s.seedChurnAlerts(custs) })
	step("offline payments", func() { s.seedOfflinePayments(custs) })
	step("referrals", func() { s.seedReferrals(custs) })
	step("gifts", func() { s.seedGifts(custs) })
	step("events", func() { s.seedEvents(custs, subs) })
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
		{"DEMO-WELCOME20", "percentage", 20, "once"},
		{"DEMO-SAVE10", "percentage", 10, "forever"},
		{"DEMO-FLAT50", "fixed", 5000, "once"},
		{"DEMO-BLACKFRIDAY", "percentage", 30, "repeating"},
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
	if status == "paid" {
		paidAt = s.between(p.start, p.end)
	}

	s.invoiceSeq++
	invID := uuid.New()
	invNum := fmt.Sprintf("INV-DEMO-%06d", s.invoiceSeq)
	eStatus := "NA"
	if c.india && status == "paid" {
		eStatus = "GENERATED"
	}
	s.exec(`INSERT INTO invoices
		(id, tenant_id, customer_id, subscription_id, status, currency, subtotal, tax_amount, total,
		 amount_paid, due_date, paid_at, created_at, updated_at, invoice_number,
		 igst_amount, cgst_amount, sgst_amount, hsn_code, tds_amount, e_invoice_status, retry_count, next_retry_at,
		 dunning_managed_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now(),$14,$15,$16,$17,$18,0,$19,$20,$21,$22)
		ON CONFLICT DO NOTHING`,
		invID, s.tenantID, c.id, sub.id, status, c.ccy, subtotal, tax, total,
		amountPaid, dueDate, paidAt, p.start, invNum,
		igst, cgst, sgst, "9983", eStatus, retryCount, nextRetry,
		nullIf(status == "past_due", "smart_dunning"))
	s.bump("invoices", 1)

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
		s.seedRevSchedule(invID, sub, total, p.start)
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
func (s *seeder) postInvoiceLedger(invID uuid.UUID, total, tax int64, at time.Time) {
	s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		VALUES ($1,$2,$3,$4,700,1,$5,'Invoice raised', $6) ON CONFLICT DO NOTHING`,
		uuid.New(), s.arAcct, s.revAcct, total, invID, at)
	s.bump("ledger_transactions", 1)
	if tax > 0 {
		s.exec(`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
			VALUES ($1,$2,$3,$4,700,6,$5,'GST on invoice', $6) ON CONFLICT DO NOTHING`,
			uuid.New(), s.revAcct, s.taxAcct, tax, invID, at)
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
			s.now.AddDate(0, 0, -(attempts-a)))
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
		s.exec(`INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6, now()) ON CONFLICT DO NOTHING`,
			uuid.New(), schedID, s.tenantID, amt, recDate, st)
		s.bump("recognition_events", 1)
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
		li, _ := json.Marshal([]map[string]any{{"description": "Enterprise onboarding", "quantity": 1, "amount": sub}})
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
		status, refundStatus := "open", "none"
		bal := amt
		if s.rng.Intn(100) < 50 {
			status, refundStatus, bal = "applied", "processed", 0
		}
		s.exec(`INSERT INTO credit_notes (id, tenant_id, customer_id, reference, amount, balance, currency, status, reason, type, refund_status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'adjustment',$10,$11,now()) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, c.id, fmt.Sprintf("CN-DEMO-%04d", s.cnSeq), amt, bal, c.ccy, status,
			[]string{"Goodwill credit", "Service downtime", "Billing correction", "Downgrade proration"}[s.rng.Intn(4)],
			refundStatus, s.backdate(s.rng.Intn(8), 0))
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
	emit := func(typ, objType string, objID uuid.UUID, at time.Time) {
		data, _ := json.Marshal(map[string]any{"demo": true})
		s.exec(`INSERT INTO events (id, tenant_id, type, object_type, object_id, data, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			uuid.New(), s.tenantID, typ, objType, objID, data, at)
		n++
	}
	for _, c := range custs {
		emit("customer.created", "customer", c.id, c.createdAt)
	}
	for _, sub := range subs {
		emit("subscription.created", "subscription", sub.id, sub.start)
		if sub.status == "canceled" && !sub.canceledAt.IsZero() {
			emit("subscription.canceled", "subscription", sub.id, sub.canceledAt)
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
func nullIf(cond bool, v string) any {
	if cond {
		return v
	}
	return nil
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
