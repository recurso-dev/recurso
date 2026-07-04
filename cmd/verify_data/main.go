package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/recur-so/recurso/internal/adapter/db"
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

	ctx := context.Background()
	tenantRepo := db.NewTenantRepository(conn.DB)

	// 1. Get Tenant by Key
	apiKey := "sk_test_12345"
	tenant, err := tenantRepo.GetTenantByKey(ctx, apiKey)
	if err != nil {
		log.Fatalf("❌ Failed to get tenant for key %s: %v", apiKey, err)
	}
	fmt.Printf("✅ Tenant Found: %s (%s)\n", tenant.Name, tenant.ID)

	// 2. Check Counts
	tables := []string{"plans", "customers", "subscriptions", "invoices", "coupons", "usage_events", "credit_notes", "quotes", "ledger_accounts", "events", "webhook_endpoints"}

	for _, t := range tables {
		var count int
		// Most tables have tenant_id. specific checks if needed.
		// events has tenant_id. usage_events doesn't have tenant_id directly?
		// usage_events has subscription_id.
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id = $1", t)
		if t == "usage_events" {
			// usage_events links to customer_id which links to tenant_id (Wait, usage_events schema?)
			// Let's check schema. Assuming it has customer_id which is enough if we join.
			// Or we just check total for now.
			query = `SELECT COUNT(*) FROM usage_events ue JOIN subscriptions s ON ue.subscription_id = s.id WHERE s.tenant_id = $1`
		}

		err := conn.GetContext(ctx, &count, query, tenant.ID)
		if err != nil {
			fmt.Printf("⚠️  Table %s: Error %v\n", t, err)
		} else {
			if count == 0 {
				fmt.Printf("❌ Table %s: 0 rows\n", t)
			} else {
				fmt.Printf("✅ Table %s: %d rows\n", t, count)
			}
		}
	}
}
