package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@127.0.0.1:5432/recurso?sslmode=disable"
	}

	conn, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer func() { _ = conn.Close() }()

	log.Println("Running Migrations (Pre-Wipe)...")
	if err := db.RunMigrations(dbURL); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("🧹 Clearing existing data...")
	tables := []string{
		"events", "webhook_endpoints", "credit_notes", "ledger_entries", "usage_events", "coupons",
		"invoices", "subscriptions", "customers", "prices", "plans", "api_keys", "tenants", "users",
	}
	for _, t := range tables {
		// Ignore errors (e.g. table doesn't exist)
		_, _ = conn.Exec("TRUNCATE TABLE " + t + " CASCADE")
	}

	log.Println("🌱 Starting Seed...")
	if err := db.RunMigrations(dbURL); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	ctx := context.Background()

	// Repositories
	// Repositories
	tenantRepo := db.NewTenantRepository(conn.DB)
	planRepo := db.NewPlanRepository(conn.DB)
	customerRepo := db.NewCustomerRepository(conn)

	log.Println("🌱 Starting Seed...")

	// 1. Create Tenant
	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:        tenantID,
		Name:      "Acme SaaS Corp",
		Email:     "admin@acmesaas.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tenantRepo.CreateTenant(ctx, tenant); err != nil {
		log.Printf("Tenant creation failed (maybe exists): %v", err)
	}
	log.Printf("Tenant: %s (%s)", tenant.Name, tenant.ID)

	// Create API Key
	apiKey := &domain.APIKey{
		ID:        uuid.New(),
		TenantID:  tenantID,
		KeyValue:  "sk_test_12345",
		Type:      "secret",
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	// Note: You may need a context with API Key or similar if CreateAPIKey requires logic,
	// but here we just pass background context.
	if err := tenantRepo.CreateAPIKey(ctx, apiKey); err != nil {
		log.Printf("API Key creation failed: %v", err)
	}

	// User creation skipped as Login uses API Key directly for demo.

	// 2. Create Plans
	plans := []struct {
		Name     string
		Code     string
		Amount   int64
		Interval string
	}{
		{"Startup", "plan_startup", 200000, "month"},    // ₹2000
		{"Business", "plan_business", 1000000, "month"}, // ₹10000
		{"Enterprise", "plan_ent", 10000000, "year"},    // ₹1 Lakh
	}

	var planIDs []uuid.UUID

	for _, p := range plans {
		planID := uuid.New() // Generate ID here to use for both
		plan := &domain.Plan{
			ID:            planID,
			TenantID:      tenantID,
			Name:          p.Name,
			Code:          p.Code,
			Active:        true,
			IntervalUnit:  domain.IntervalUnit("month"),
			IntervalCount: 1,
			CreatedAt:     time.Now(),
			Prices: []domain.Price{
				{
					ID:        uuid.New(),
					PlanID:    planID, // Fix: Link to Plan
					Currency:  "INR",
					Amount:    p.Amount,
					Type:      "recurring",
					CreatedAt: time.Now(),
				},
			},
		}
		if p.Interval == "year" {
			plan.IntervalUnit = domain.IntervalUnit("year")
		}

		if err := planRepo.Create(ctx, plan); err != nil {
			log.Printf("Plan %s exists or failed: %v", p.Name, err)
		}
		planIDs = append(planIDs, plan.ID)
		log.Printf("Plan Created: %s", p.Name)
	}

	// 3. Create Customers & Subscriptions
	names := []string{"Alpha Corp", "Beta Ltd", "Gamma Inc", "Delta LLC", "John Doe", "Jane Smith", "Bob Wilson", "Alice Brown"}

	for i := 0; i < 20; i++ {
		custID := uuid.New()
		name := fmt.Sprintf("Customer %d", i+1)
		if i < len(names) {
			name = names[i]
		}

		isB2B := i%3 == 0 // Every 3rd is B2B
		taxType := "consumer"
		gstin := ""
		if isB2B {
			taxType = "business"
			gstin = fmt.Sprintf("29ABCDE%04dF1Z5", i)
		}

		cust := &domain.Customer{
			ID:       custID,
			TenantID: tenantID,
			Name:     domain.StringPtr(name),
			Email:    fmt.Sprintf("customer%d@example.com", i),
			Phone:    "+919876543210",
			TaxType:  taxType,
			GSTIN:    domain.PtrStringPtr(gstin),
			BillingAddress: domain.BillingAddress{
				Line1:   "123 Tech Park",
				City:    "Bengaluru",
				State:   "Karnataka",
				Country: "India",
				Zip:     "560001",
			},
			CreatedAt: time.Now(),
		}
		if err := customerRepo.Create(ctx, cust); err != nil {
			log.Printf("Failed cust: %v", err)
			continue
		}

		// Create Subscription (Raw SQL for speed/bypassing service logic which might require GSP calls)
		// Ensure we have valid plans
		if len(planIDs) == 0 {
			continue
		}
		planID := planIDs[rand.Intn(len(planIDs))]
		status := domain.SubscriptionStatusActive
		if i > 15 {
			status = domain.SubscriptionStatusCanceled
		} else if i > 13 {
			status = domain.SubscriptionStatusPastDue
		}

		subID := uuid.New()
		start := time.Now().AddDate(0, -rand.Intn(6), 0) // Started 0-6 months ago

		// Raw Insert Subscription to update status easily
		_, err = conn.Exec(`
            INSERT INTO subscriptions (
                id, tenant_id, customer_id, plan_id, status, 
                current_period_start, current_period_end, billing_anchor, 
                created_at, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        `, subID, tenantID, custID, planID, status,
			start, start.AddDate(0, 1, 0), start,
			start, time.Now())

		if err != nil {
			log.Printf("Failed sub: %v", err)
		}

		// Create an Invoice
		invID := uuid.New()
		amount := int64(200000) // Mock

		// Mock IRN if B2B
		irn := ""
		e_status := "PENDING" // Fix: Valid ENUM
		if isB2B {
			irn = "mock_irn_" + uuid.New().String()
			e_status = "GENERATED"
		}

		_, err = conn.Exec(`
            INSERT INTO invoices (
                id, tenant_id, subscription_id, customer_id, invoice_number,
                status, currency, subtotal, tax_amount, total,
                irn, e_invoice_status,
                created_at, due_date
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        `, invID, tenantID, subID, custID, fmt.Sprintf("INV-%d", i+1000),
			"paid", "INR", amount, 36000, amount+36000,
			irn, e_status,
			start, start) // Created when sub started

		if err != nil {
			log.Printf("Failed inv: %v", err)
		}
	}

	// =========================================================================
	// 4. Seed Extras (Coupons, Usage, Credit Notes, Quotes, Ledger)
	// =========================================================================

	// Reuse last IDs (Warning: if loop didn't run, these might be empty/nil, but loop runs 20 times)
	// Actually, we define seed variables here by querying or just using the last created ones if we captured them.
	// But we didn't capture them in the loop above.
	// Let's create a *new* specific customer for these extras to be safe and clean.

	extraCustID := uuid.New()
	extraCust := &domain.Customer{
		ID:        extraCustID,
		TenantID:  tenantID,
		Name:      domain.StringPtr("Extra Features Demo Ltd"),
		Email:     "extra@demo.com",
		CreatedAt: time.Now(),
	}
	if err := customerRepo.Create(ctx, extraCust); err != nil {
		log.Printf("Failed to create extra customer: %v", err)
	}

	// --- Coupons ---
	couponID := uuid.New()
	if _, err := conn.Exec(`INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		couponID, tenantID, "WELCOME20", "percent", 20, "forever"); err != nil {
		log.Printf("Failed to create coupon 1: %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		uuid.New(), tenantID, "FLAT100", "amount", 10000, "once"); err != nil {
		log.Printf("Failed to create coupon 2: %v", err)
	}

	log.Println("🎟️  Coupons Created")

	// --- Usage ---
	// Need a subscription for this customer
	usageSubID := uuid.New()
	// Insert dummy sub
	if _, err := conn.Exec(`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW() + interval '1 month', NOW(), NOW(), NOW())`,
		usageSubID, tenantID, extraCustID, planIDs[0], "active"); err != nil {
		log.Printf("Failed usage sub: %v", err)
	}

	// Insert Usage Events
	if _, err := conn.Exec(`INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp)
		VALUES ($1, $2, $3, $4, $5, NOW())`,
		uuid.New(), usageSubID, extraCustID, "api_calls", 500); err != nil {
		log.Printf("Failed usage event 1: %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO usage_events (id, subscription_id, customer_id, dimension, quantity, timestamp)
		VALUES ($1, $2, $3, $4, $5, NOW() - interval '1 day')`,
		uuid.New(), usageSubID, extraCustID, "api_calls", 120); err != nil {
		log.Printf("Failed usage event 2: %v", err)
	}

	log.Println("📊 Usage Events Created")

	// --- Credit Notes ---
	if _, err := conn.Exec(`INSERT INTO credit_notes (id, tenant_id, customer_id, amount, balance, currency, status, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		uuid.New(), tenantID, extraCustID, 5000, 5000, "USD", "issued", "Service Downtime Compensation"); err != nil {
		log.Printf("Failed to create credit note: %v", err)
	}

	log.Println("💵 Credit Notes Created")

	// --- Quotes ---
	if _, err := conn.Exec(`INSERT INTO quotes (id, tenant_id, customer_id, quote_number, status, subtotal, total, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		uuid.New(), tenantID, extraCustID, "QT-1001", "draft", 50000, 50000, "USD"); err != nil {
		log.Printf("Failed to create quote 1: %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO quotes (id, tenant_id, customer_id, quote_number, status, subtotal, total, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW() - interval '2 days')`,
		uuid.New(), tenantID, extraCustID, "QT-0999", "accepted", 120000, 120000, "USD"); err != nil {
		log.Printf("Failed to create quote 2: %v", err)
	}

	log.Println("📜 Quotes Created")

	// --- Ledger Accounts ---
	// 1000: Assets, 4000: Revenue
	if _, err := conn.Exec(`INSERT INTO ledger_accounts (id, tenant_id, name, type, code, ledger_id, currency) VALUES
		($1, $2, 'Cash', 'asset', 1001, 1, 'USD'),
		($3, $2, 'Accounts Receivable', 'asset', 1002, 1, 'USD'),
		($4, $2, 'Subscription Revenue', 'revenue', 4001, 1, 'USD')`,
		uuid.New(), tenantID, uuid.New(), uuid.New()); err != nil {
		log.Printf("Failed to create ledger accounts: %v", err)
	}

	log.Println("📒 Ledger Accounts Created")

	// --- Events (for Developers -> Event Logs) ---
	if _, err := conn.Exec(`INSERT INTO events (id, tenant_id, type, object_id, object_type, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() - interval '1 hour')`,
		uuid.New(), tenantID, "customer.created", extraCustID, "customer", "{}"); err != nil {
		log.Printf("Failed event 1: %v", err)
	}

	if _, err := conn.Exec(`INSERT INTO events (id, tenant_id, type, object_id, object_type, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		uuid.New(), tenantID, "invoice.created", uuid.New(), "invoice", "{}"); err != nil {
		log.Printf("Failed event 2: %v", err)
	}

	log.Println("📜 Events Created")

	// --- Webhooks (for Developers -> Webhooks) ---
	// Need a secret
	whSecret := "whsec_" + uuid.New().String()
	// FIX: Table Name is webhook_endpoints
	if _, err := conn.Exec(`INSERT INTO webhook_endpoints (id, tenant_id, url, secret, events, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		uuid.New(), tenantID, "https://workflow.example.com/webhook", whSecret,
		`{"invoice.paid", "subscription.canceled"}`, "active"); err != nil {
		log.Printf("Failed webhook: %v", err)
	}

	log.Println("🪝 Webhooks Created")

	log.Println("✅ Seeding Complete!")
}
