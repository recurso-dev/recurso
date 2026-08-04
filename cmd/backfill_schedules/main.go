// Command backfill_schedules creates revenue-recognition schedules at issuance
// for a tenant's EXISTING open subscription invoices — the operational step that
// completes an accrual rollout (#466). Enabling RECURSO_ACCRUAL_RECOGNITION only
// schedules NEW invoices at issuance; a tenant's already-open invoices keep their
// cash-model (no) schedule, so their deferred stays in the close pack's
// awaiting-payment bucket until backfilled. After this runs, that deferred moves
// into the scheduled bucket and the month-end tie-out reads zero.
//
// Dry-run by default (reports what it would do); pass --apply to write. Idempotent
// (CreateScheduleForInvoice is a no-op for an invoice that already has a schedule),
// so it is safe to re-run.
//
//	DATABASE_URL=... go run ./cmd/backfill_schedules --tenant=<uuid>          # dry run
//	DATABASE_URL=... go run ./cmd/backfill_schedules --tenant=<uuid> --apply  # execute
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

func main() {
	tenantFlag := flag.String("tenant", "", "tenant UUID to backfill (required)")
	apply := flag.Bool("apply", false, "write schedules (default: dry run, report only)")
	flag.Parse()

	tenantID, err := uuid.Parse(*tenantFlag)
	if err != nil {
		log.Fatalf("--tenant must be a valid UUID: %v", err)
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Tenant on the context so the tenant-scoped repositories filter correctly.
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)

	ledger := service.NewLedgerService(nil, db.NewLedgerRepository(conn))
	subRepo := db.NewSubscriptionRepository(conn)
	revrec := service.NewRevRecService(db.NewRevRecRepository(conn), ledger, subRepo)
	invoiceRepo := db.NewInvoiceRepository(conn)

	// Eligible: subscription invoices that are still deferred (open/past_due) and
	// have no active recognition schedule. Same shape the close pack's
	// awaiting-payment bucket counts, minus the already-scheduled ones.
	rows, err := conn.QueryContext(ctx, `
		SELECT i.id, COALESCE(i.subtotal,0)
		FROM invoices i
		WHERE i.tenant_id = $1
		  AND i.subscription_id IS NOT NULL
		  AND i.status IN ('open','past_due')
		  AND NOT EXISTS (
		    SELECT 1 FROM revenue_schedules s
		    WHERE s.invoice_id = i.id AND s.status = 'active')
		ORDER BY i.created_at`, tenantID)
	if err != nil {
		log.Fatalf("query eligible invoices: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type target struct {
		id       uuid.UUID
		subtotal int64
	}
	var targets []target
	var totalDeferred int64
	for rows.Next() {
		var tt target
		if err := rows.Scan(&tt.id, &tt.subtotal); err != nil {
			log.Fatalf("scan: %v", err)
		}
		targets = append(targets, tt)
		totalDeferred += tt.subtotal
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	fmt.Printf("backfill_schedules [%s] tenant=%s\n", mode, tenantID)
	fmt.Printf("eligible open subscription invoices without a schedule: %d (deferred ≈ %d minor units)\n",
		len(targets), totalDeferred)

	if !*apply {
		fmt.Println("dry run — no schedules written. Re-run with --apply to create them.")
		return
	}

	var scheduled, failed int
	for _, tt := range targets {
		inv, err := invoiceRepo.GetByID(ctx, tt.id)
		if err != nil || inv == nil {
			log.Printf("  skip %s: load failed: %v", tt.id, err)
			failed++
			continue
		}
		// sub=nil → CreateScheduleForInvoice resolves it from the subscription
		// repo; it is idempotent per invoice.
		if err := revrec.CreateScheduleForInvoice(ctx, inv, nil); err != nil {
			log.Printf("  fail %s: %v", tt.id, err)
			failed++
			continue
		}
		scheduled++
	}
	fmt.Printf("done: %d scheduled, %d failed. Re-run is safe (idempotent).\n", scheduled, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
