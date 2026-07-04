package main

import (
	"context"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/recurso?sslmode=disable"
	}

	dbx, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	database := dbx.DB

	// Repos
	invoiceRepo := db.NewInvoiceRepository(database)
	revrecRepo := db.NewRevRecRepository(database)

	subscriptionRepo := db.NewSubscriptionRepository(database)

	// Services
	ledgerService := service.NewLedgerService(nil, nil) // Mock TB + PG for now
	revrecService := service.NewRevRecService(revrecRepo, ledgerService, subscriptionRepo)

	// Get a sample invoice
	var invID, tenantIDStr string
	err = database.QueryRow("SELECT id, tenant_id FROM invoices LIMIT 1").Scan(&invID, &tenantIDStr)
	if err != nil {
		log.Fatalf("failed to get sample invoice: %v", err)
	}

	log.Printf("Found sample invoice: %s (Tenant: %s)", invID, tenantIDStr)

	tenantID := uuid.MustParse(tenantIDStr)
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)

	inv, err := invoiceRepo.GetByID(ctx, uuid.MustParse(invID))
	if err != nil {
		log.Fatalf("failed to get invoice: %v", err)
	}

	// Trigger RevRec manually
	log.Printf("Creating RevRec schedule for invoice %s...", inv.InvoiceNumber)
	if err := revrecService.CreateScheduleForInvoice(ctx, inv, nil); err != nil {
		log.Fatalf("failed to create schedule: %v", err)
	}

	log.Println("✅ Revenue Recognition schedule created successfully!")

	// Verify in DB
	var count int
	if err := database.QueryRow("SELECT count(*) FROM revenue_schedules WHERE invoice_id = $1", inv.ID).Scan(&count); err != nil {
		log.Fatalf("failed to count schedules: %v", err)
	}
	log.Printf("Schedules in DB: %d", count)

	if err := database.QueryRow("SELECT count(*) FROM recognition_events WHERE tenant_id = $1", inv.TenantID).Scan(&count); err != nil {
		log.Fatalf("failed to count recognition events: %v", err)
	}
	log.Printf("Total Recognition Events in DB: %d", count)

	// Test processing
	log.Println("Processing due recognition events...")
	if err := revrecService.ProcessDueEvents(ctx); err != nil {
		log.Fatalf("failed to process due events: %v", err)
	}

	if err := database.QueryRow("SELECT count(*) FROM recognition_events WHERE tenant_id = $1 AND status = 'recognized'", inv.TenantID).Scan(&count); err != nil {
		log.Fatalf("failed to count recognized events: %v", err)
	}
	log.Printf("Recognized Events in DB: %d", count)

	if count > 0 {
		log.Println("✅ Revenue Recognition processing verified!")
	} else {
		log.Fatal("❌ No events were recognized")
	}
}
