// Command import loads customers, plans, and subscriptions from another
// billing system into Recurso without generating invoices or touching
// payment gateways — safe for migrating a live subscriber base.
//
// Unlike API-driven creation, imported subscriptions keep their original
// billing period; the renewal worker issues the next invoice at
// current_period_end, so migrated customers are not double-billed.
//
// Idempotent: plans match by code, customers by email, subscriptions by
// external_id. Re-running an import skips existing records.
//
// Usage:
//
//	go run ./cmd/import -tenant <tenant-uuid> -input data.json [-dry-run]
//	go run ./cmd/import -tenant <tenant-uuid> \
//	    -plans-csv plans.csv -customers-csv customers.csv \
//	    -subscriptions-csv subs.csv [-dry-run]
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
		conn:     conn,
		tenantID: tenantID,
		dryRun:   *dryRun,
		planRepo: db.NewPlanRepository(conn.DB),
		custRepo: db.NewCustomerRepository(conn),
		subRepo:  db.NewSubscriptionRepository(conn.DB),
		ledger:   db.NewLedgerRepository(conn.DB),
	}

	ctx := context.Background()
	if err := imp.verifyTenant(ctx); err != nil {
		log.Fatal(err)
	}

	if *dryRun {
		log.Println("DRY RUN — no changes will be written")
	}
	imp.importPlans(ctx, file.Plans)
	imp.importCustomers(ctx, file.Customers)
	imp.importSubscriptions(ctx, file.Subscriptions)

	log.Printf("Done: %d created, %d skipped (already exist), %d failed",
		imp.created, imp.skipped, imp.failed)
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
	conn     *sqlx.DB
	tenantID uuid.UUID
	dryRun   bool

	planRepo port.PlanRepository
	custRepo *db.CustomerRepository
	subRepo  port.SubscriptionRepository
	ledger   *db.LedgerRepository

	created int
	skipped int
	failed  int

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
		existing, err := im.planRepo.GetByCode(ctx, im.tenantID, p.Code)
		if err == nil && existing != nil {
			im.planIDs[p.Code] = existing.ID
			im.skipped++
			log.Printf("plan %s: exists, skipping", p.Code)
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
		var existingID uuid.UUID
		err := im.conn.QueryRowContext(ctx,
			`SELECT id FROM customers WHERE tenant_id = $1 AND lower(email) = $2`,
			im.tenantID, key).Scan(&existingID)
		if err == nil {
			im.customerIDs[key] = existingID
			im.skipped++
			log.Printf("customer %s: exists, skipping", c.Email)
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
		var existingID uuid.UUID
		err := im.conn.QueryRowContext(ctx,
			`SELECT id FROM subscriptions WHERE tenant_id = $1 AND reference_id = $2`,
			im.tenantID, s.ExternalID).Scan(&existingID)
		if err == nil {
			im.skipped++
			log.Printf("subscription %s: exists, skipping", s.ExternalID)
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
			existing, perr := im.planRepo.GetByCode(ctx, im.tenantID, s.PlanCode)
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
