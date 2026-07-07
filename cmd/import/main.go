// Command import loads customers, plans, and subscriptions from another
// billing system into Recurso without generating invoices or touching
// payment gateways — safe for migrating a live subscriber base.
//
// Unlike API-driven creation, imported subscriptions keep their original
// billing period; the renewal worker issues the next invoice at
// current_period_end, so migrated customers are not double-billed.
//
// Idempotent: plans match by code, customers by email, subscriptions by
// external_id. Re-running an import skips existing records by default.
//
// Two bulk modes change what happens to records that already exist, and both
// honor -dry-run (report the actions without writing anything):
//
//   - -update rewrites the provider-safe fields of matched records in place
//     (customer name/country/GST, plan name/amount, subscription status/period)
//     instead of skipping them. Matching is unchanged, so it never duplicates.
//
//   - -cancel-missing (cancel-sync) makes the import authoritative: any
//     subscription that exists for the tenant but is absent from the file is
//     scheduled for period-end cancellation. It only ever touches import-origin
//     subscriptions (those with an external_id) — dashboard-created ones are
//     left alone — and refuses to run when the file lists no subscriptions.
//
// Usage:
//
//	go run ./cmd/import -tenant <tenant-uuid> -input data.json [-dry-run]
//	go run ./cmd/import -tenant <tenant-uuid> \
//	    -plans-csv plans.csv -customers-csv customers.csv \
//	    -subscriptions-csv subs.csv [-dry-run]
//
//	# update existing records in place instead of skipping them:
//	go run ./cmd/import -tenant <tenant-uuid> -input data.json -update [-dry-run]
//
//	# make the file authoritative — cancel import-origin subs it omits
//	# (always dry-run first to see exactly what would be canceled):
//	go run ./cmd/import -tenant <tenant-uuid> -input data.json -cancel-missing -dry-run
//	go run ./cmd/import -tenant <tenant-uuid> -input data.json -update -cancel-missing
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

func main() {
	var (
		tenantFlag = flag.String("tenant", "", "tenant UUID to import into (required; register the tenant first)")
		inputFlag  = flag.String("input", "", "JSON import file (see cmd/import/example.json)")
		plansCSV   = flag.String("plans-csv", "", "plans CSV (code,name,amount,currency,interval_unit,interval_count)")
		custCSV    = flag.String("customers-csv", "", "customers CSV (email,name,country,...)")
		subsCSV    = flag.String("subscriptions-csv", "", "subscriptions CSV (external_id,customer_email,plan_code,status,current_period_start,current_period_end)")
		dryRun     = flag.Bool("dry-run", false, "validate and report actions without writing anything")
		update     = flag.Bool("update", false, "update matched records in place (provider-safe fields) instead of skipping them")
		cancelMiss = flag.Bool("cancel-missing", false, "cancel-sync: schedule period-end cancellation for import-origin subscriptions absent from the file (authoritative import)")
	)
	flag.Parse()

	if *tenantFlag == "" {
		log.Fatal("-tenant is required (find it in the dashboard under Settings, or in the /auth/register response)")
	}
	tenantID, err := uuid.Parse(*tenantFlag)
	if err != nil {
		log.Fatalf("-tenant is not a valid UUID: %v", err)
	}

	file, err := loadInput(*inputFlag, *plansCSV, *custCSV, *subsCSV)
	if err != nil {
		log.Fatal(err)
	}
	if errs := file.Validate(); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("VALIDATION: %v", e)
		}
		log.Fatalf("import file invalid: %d problem(s) — nothing was written", len(errs))
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@127.0.0.1:5432/recurso?sslmode=disable"
	}
	conn, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() { _ = conn.Close() }()

	imp := &importer{
		conn:          conn,
		tenantID:      tenantID,
		dryRun:        *dryRun,
		update:        *update,
		cancelMissing: *cancelMiss,
		store:         &dbStore{db: conn},
		planRepo:      db.NewPlanRepository(conn.DB),
		custRepo:      db.NewCustomerRepository(conn),
		subRepo:       db.NewSubscriptionRepository(conn.DB),
		ledger:        db.NewLedgerRepository(conn.DB),
	}

	ctx := context.Background()
	if err := imp.verifyTenant(ctx); err != nil {
		log.Fatal(err)
	}

	if *dryRun {
		log.Println("DRY RUN — no changes will be written")
	}
	if *update {
		log.Println("UPDATE mode — matched records will be updated in place, not skipped")
	}
	imp.importPlans(ctx, file.Plans)
	imp.importCustomers(ctx, file.Customers)
	imp.importSubscriptions(ctx, file.Subscriptions)
	if *cancelMiss {
		imp.cancelMissingSubscriptions(ctx, file.Subscriptions)
	}

	log.Printf("Done: %d created, %d updated, %d skipped, %d canceled, %d failed",
		imp.created, imp.updated, imp.skipped, imp.canceled, imp.failed)
	if !*dryRun && imp.created > 0 {
		log.Println("No invoices were generated. The renewal worker will issue each")
		log.Println("subscription's next invoice at its current_period_end.")
	}
	if imp.failed > 0 {
		os.Exit(1)
	}
}

func loadInput(input, plansCSV, custCSV, subsCSV string) (*ImportFile, error) {
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		return ParseJSON(f)
	}
	if custCSV == "" && subsCSV == "" && plansCSV == "" {
		return nil, fmt.Errorf("provide -input file.json, or CSV files via -plans-csv/-customers-csv/-subscriptions-csv")
	}
	file := &ImportFile{}
	if plansCSV != "" {
		f, err := os.Open(plansCSV)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		if file.Plans, err = ParsePlansCSV(f); err != nil {
			return nil, err
		}
	}
	if custCSV != "" {
		f, err := os.Open(custCSV)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		if file.Customers, err = ParseCustomersCSV(f); err != nil {
			return nil, err
		}
	}
	if subsCSV != "" {
		f, err := os.Open(subsCSV)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		if file.Subscriptions, err = ParseSubscriptionsCSV(f); err != nil {
			return nil, err
		}
	}
	return file, nil
}

type importer struct {
	conn          *sqlx.DB
	tenantID      uuid.UUID
	dryRun        bool
	update        bool // -update: update matched records instead of skipping
	cancelMissing bool // -cancel-missing: cancel-sync import-origin subs absent from the file

	store    store
	planRepo port.PlanRepository
	custRepo *db.CustomerRepository
	subRepo  port.SubscriptionRepository
	ledger   *db.LedgerRepository

	created  int
	updated  int
	skipped  int
	canceled int
	failed   int

	// resolved during the run, used to link subscriptions
	planIDs     map[string]uuid.UUID // plan code -> id
	customerIDs map[string]uuid.UUID // lower(email) -> id
}

func (im *importer) verifyTenant(ctx context.Context) error {
	var name string
	err := im.conn.QueryRowContext(ctx, `SELECT name FROM tenants WHERE id = $1`, im.tenantID).Scan(&name)
	if err != nil {
		return fmt.Errorf("tenant %s not found — register it first (POST /auth/register): %w", im.tenantID, err)
	}
	log.Printf("Importing into tenant %q (%s)", name, im.tenantID)
	return nil
}

func (im *importer) importPlans(ctx context.Context, plans []PlanInput) {
	im.planIDs = map[string]uuid.UUID{}
	for _, p := range plans {
		existing, err := im.store.PlanByCode(ctx, im.tenantID, p.Code)
		if err != nil {
			im.failed++
			log.Printf("plan %s: FAILED: %v", p.Code, err)
			continue
		}
		if existing != nil {
			im.planIDs[p.Code] = existing.ID
			if !im.update {
				im.skipped++
				log.Printf("plan %s: exists, skipping", p.Code)
				continue
			}
			if im.dryRun {
				im.updated++
				log.Printf("plan %s: would update (name %q, %d %s)", p.Code, p.Name, p.Amount, p.Currency)
				continue
			}
			if err := im.store.UpdatePlan(ctx, im.tenantID, *existing, p); err != nil {
				im.failed++
				log.Printf("plan %s: FAILED: %v", p.Code, err)
				continue
			}
			im.updated++
			log.Printf("plan %s: updated", p.Code)
			continue
		}
		if im.dryRun {
			im.created++
			log.Printf("plan %s: would create (%d %s / %d %s)", p.Code, p.Amount, p.Currency, p.IntervalCount, p.IntervalUnit)
			continue
		}
		plan := &domain.Plan{
			ID:            uuid.New(),
			TenantID:      im.tenantID,
			Name:          p.Name,
			Code:          p.Code,
			IntervalUnit:  domain.IntervalUnit(p.IntervalUnit),
			IntervalCount: p.IntervalCount,
			Active:        true,
			CreatedAt:     time.Now().UTC(),
			Prices: []domain.Price{{
				ID:       uuid.New(),
				Currency: strings.ToUpper(p.Currency),
				Amount:   p.Amount,
				Type:     "recurring",
			}},
		}
		if err := im.planRepo.Create(ctx, plan); err != nil {
			im.failed++
			log.Printf("plan %s: FAILED: %v", p.Code, err)
			continue
		}
		im.planIDs[p.Code] = plan.ID
		im.created++
		log.Printf("plan %s: created", p.Code)
	}
}

func (im *importer) importCustomers(ctx context.Context, customers []CustomerInput) {
	im.customerIDs = map[string]uuid.UUID{}
	for _, c := range customers {
		key := strings.ToLower(c.Email)
		existingID, found, err := im.store.CustomerIDByEmail(ctx, im.tenantID, c.Email)
		if err != nil {
			im.failed++
			log.Printf("customer %s: FAILED: %v", c.Email, err)
			continue
		}
		if found {
			im.customerIDs[key] = existingID
			if !im.update {
				im.skipped++
				log.Printf("customer %s: exists, skipping", c.Email)
				continue
			}
			if im.dryRun {
				im.updated++
				log.Printf("customer %s: would update (name %q, country %q)", c.Email, c.Name, c.Country)
				continue
			}
			if err := im.store.UpdateCustomer(ctx, im.tenantID, existingID, c); err != nil {
				im.failed++
				log.Printf("customer %s: FAILED: %v", c.Email, err)
				continue
			}
			im.updated++
			log.Printf("customer %s: updated", c.Email)
			continue
		}
		if im.dryRun {
			im.created++
			log.Printf("customer %s: would create", c.Email)
			continue
		}
		cust := &domain.Customer{
			ID:       uuid.New(),
			TenantID: im.tenantID,
			Email:    c.Email,
			Name:     domain.StringPtr(c.Name),
			Phone:    c.Phone,
			BillingAddress: domain.BillingAddress{
				Line1:   c.Line1,
				City:    c.City,
				State:   c.State,
				Zip:     c.Zip,
				Country: c.Country,
			},
			CreatedAt: time.Now().UTC(),
		}
		if c.TaxID != "" {
			cust.TaxID = domain.StringPtr(c.TaxID)
		}
		if c.GSTIN != "" {
			cust.GSTIN = domain.StringPtr(c.GSTIN)
		}
		if c.PlaceOfSupply != "" {
			cust.PlaceOfSupply = domain.StringPtr(c.PlaceOfSupply)
		}
		if err := im.custRepo.Create(ctx, cust); err != nil {
			im.failed++
			log.Printf("customer %s: FAILED: %v", c.Email, err)
			continue
		}
		// AR sub-ledger, same as service.LedgerService.CreateCustomerAccounts
		// (CreateAccount is ON CONFLICT DO NOTHING, so this is idempotent too).
		_ = im.ledger.CreateAccount(ctx, &domain.LedgerAccount{
			ID:       cust.ID,
			TenantID: im.tenantID,
			Name:     "Accounts Receivable",
			Type:     domain.AccountTypeAsset,
			Code:     domain.AccountCodeAR,
			LedgerID: 1,
		})
		im.customerIDs[key] = cust.ID
		im.created++
		log.Printf("customer %s: created", c.Email)
	}
}

func (im *importer) importSubscriptions(ctx context.Context, subs []SubscriptionInput) {
	for _, s := range subs {
		existingID, found, err := im.store.SubscriptionIDByExternalID(ctx, im.tenantID, s.ExternalID)
		if err != nil {
			im.failed++
			log.Printf("subscription %s: FAILED: %v", s.ExternalID, err)
			continue
		}
		if found {
			if !im.update {
				im.skipped++
				log.Printf("subscription %s: exists, skipping", s.ExternalID)
				continue
			}
			if im.dryRun {
				im.updated++
				log.Printf("subscription %s: would update (status %s, period %s..%s)", s.ExternalID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd)
				continue
			}
			if err := im.store.UpdateSubscription(ctx, im.tenantID, existingID, s); err != nil {
				im.failed++
				log.Printf("subscription %s: FAILED: %v", s.ExternalID, err)
				continue
			}
			im.updated++
			log.Printf("subscription %s: updated", s.ExternalID)
			continue
		}

		customerID, ok := im.customerIDs[strings.ToLower(s.CustomerEmail)]
		if !ok {
			im.failed++
			log.Printf("subscription %s: FAILED: customer %s was not imported", s.ExternalID, s.CustomerEmail)
			continue
		}
		planID, ok := im.planIDs[s.PlanCode]
		if !ok {
			// Plan not in this import file; it must already exist.
			existing, perr := im.store.PlanByCode(ctx, im.tenantID, s.PlanCode)
			if perr != nil || existing == nil {
				im.failed++
				log.Printf("subscription %s: FAILED: plan code %q not found in tenant", s.ExternalID, s.PlanCode)
				continue
			}
			planID = existing.ID
			im.planIDs[s.PlanCode] = planID
		}
		if im.dryRun {
			im.created++
			log.Printf("subscription %s: would create (%s on %s, %s)", s.ExternalID, s.CustomerEmail, s.PlanCode, s.Status)
			continue
		}

		start, _ := time.Parse(time.RFC3339, s.CurrentPeriodStart) // validated earlier
		end, _ := time.Parse(time.RFC3339, s.CurrentPeriodEnd)
		now := time.Now().UTC()
		sub := &domain.Subscription{
			ID:                 uuid.New(),
			TenantID:           im.tenantID,
			CustomerID:         customerID,
			PlanID:             planID,
			Status:             domain.SubscriptionStatus(s.Status),
			CurrentPeriodStart: start,
			CurrentPeriodEnd:   end,
			BillingAnchor:      start,
			PaymentTerms:       s.PaymentTerms,
			ReferenceID:        s.ExternalID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := im.subRepo.Create(ctx, sub); err != nil {
			im.failed++
			log.Printf("subscription %s: FAILED: %v", s.ExternalID, err)
			continue
		}
		im.created++
		log.Printf("subscription %s: created (renews %s)", s.ExternalID, end.Format("2006-01-02"))
	}
}

// cancelMissingSubscriptions implements cancel-sync (-cancel-missing): it makes
// the import file authoritative by scheduling a period-end cancellation for any
// subscription that exists for the tenant but is absent from the file.
//
// Safety guards:
//   - Only import-origin subscriptions (a non-empty reference_id / external_id)
//     are considered; dashboard-created subscriptions are never touched.
//   - When the file lists no subscriptions it refuses to run, so a truncated
//     export can't wipe the whole tenant.
//   - In dry-run every candidate is logged but nothing is written.
func (im *importer) cancelMissingSubscriptions(ctx context.Context, subs []SubscriptionInput) {
	if len(subs) == 0 {
		log.Println("cancel-sync: SKIPPED — the import file lists no subscriptions; refusing to cancel the entire tenant. Provide the full authoritative subscription set.")
		return
	}

	present := make(map[string]bool, len(subs))
	for _, s := range subs {
		present[s.ExternalID] = true
	}

	existing, err := im.store.ListSubscriptions(ctx, im.tenantID)
	if err != nil {
		im.failed++
		log.Printf("cancel-sync: FAILED to list subscriptions: %v", err)
		return
	}

	for _, e := range existing {
		if e.ReferenceID == "" {
			// Dashboard-created subscription (no external_id) — never touch it.
			continue
		}
		if present[e.ReferenceID] {
			continue // still in the authoritative file; keep it.
		}
		if im.dryRun {
			im.canceled++
			log.Printf("subscription %s: WOULD CANCEL (absent from import, period-end)", e.ReferenceID)
			continue
		}
		if err := im.store.CancelSubscription(ctx, im.tenantID, e.ID, cancelSyncReason); err != nil {
			im.failed++
			log.Printf("subscription %s: FAILED to cancel: %v", e.ReferenceID, err)
			continue
		}
		im.canceled++
		log.Printf("subscription %s: canceled at period-end (absent from import)", e.ReferenceID)
	}
}
