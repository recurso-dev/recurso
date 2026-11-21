package service

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestManualInvoice_Postgres drives the manual-invoice service against real
// Postgres: per-entity gapless numbering, signed lines, ledger posting that
// the reconciler accepts, net-zero handling, dunning exclusion and the
// external_ref idempotency guarantee (including under concurrency).
func TestManualInvoice_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed manual-invoice test")
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
	ctx := context.Background()
	run := uuid.NewString()[:8]

	seedTenant := func(label string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
			id, label+"-"+run, label+"-"+run+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
		return id
	}
	seedCustomer := func(tenantID uuid.UUID, name string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO customers (id, tenant_id, email, name, country, tax_type, ledger_account_id, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,' ','individual',$5,NOW(),NOW())`,
			id, tenantID, name+"-"+run+"@example.invalid", name, uuid.New()); err != nil {
			t.Fatalf("seed customer: %v", err)
		}
		return id
	}

	tenantID := seedTenant("MI")
	otherTenant := seedTenant("MI-OTHER")
	entityRepo := db.NewEntityRepository(conn)
	duk := &domain.Entity{TenantID: tenantID, Name: "UKCO", LegalName: "UKCO Ltd", InvoicePrefix: "UKCO"}
	if err := entityRepo.Create(ctx, duk); err != nil {
		t.Fatalf("create UKCO entity: %v", err)
	}
	dsg := &domain.Entity{TenantID: tenantID, Name: "SGCO", LegalName: "SGCO Ltd", InvoicePrefix: "SGCO"}
	if err := entityRepo.Create(ctx, dsg); err != nil {
		t.Fatalf("create SGCO entity: %v", err)
	}
	custA := seedCustomer(tenantID, "Acme")
	foreignCust := seedCustomer(otherTenant, "Foreign")

	invRepo := db.NewInvoiceRepository(conn).(*db.InvoiceRepository)
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)
	svc := NewManualInvoiceService(invRepo, db.NewCustomerRepository(dbx), entityRepo, db.NewTxManager(conn))
	svc.SetLedgerPoster(ledger)

	aug1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	lines := func(l ...ManualInvoiceLine) []ManualInvoiceLine { return l }
	newIn := func(entity *domain.Entity, ref string, ls []ManualInvoiceLine) CreateManualInvoiceInput {
		in := CreateManualInvoiceInput{
			TenantID: tenantID, CustomerID: custA, Currency: "USD",
			InvoiceDate: aug1, ExternalRef: ref, Lines: ls,
		}
		if entity != nil {
			in.EntityID = &entity.ID
		}
		return in
	}
	feeAndWaiver := lines(
		ManualInvoiceLine{Description: "Data feed", Amount: 1500_00},
		ManualInvoiceLine{Description: "Connection fee", Amount: 500_00},
		ManualInvoiceLine{Description: "Data feed waiver", Amount: -700_00},
	)

	// 1. Fee + waiver on the UKCO entity.
	inv1, created, err := svc.CreateManualInvoice(ctx, newIn(duk, "cust-a|UKCO|2026-08", feeAndWaiver))
	if err != nil || !created {
		t.Fatalf("create #1: created=%v err=%v", created, err)
	}
	if inv1.InvoiceNumber != "UKCO-000001" {
		t.Errorf("number = %q, want UKCO-000001 (the issuing entity's own series)", inv1.InvoiceNumber)
	}
	got1, err := invRepo.GetByID(context.WithValue(ctx, domain.TenantIDKey, tenantID), inv1.ID)
	if err != nil {
		t.Fatalf("reload #1: %v", err)
	}
	if got1.Total != 1300_00 || got1.Status != domain.InvoiceStatusOpen {
		t.Errorf("#1 total %d status %q, want 130000 / open", got1.Total, got1.Status)
	}
	if len(got1.LineItems) != 3 {
		t.Fatalf("#1 persisted %d lines, want 3 (waiver kept as its own line)", len(got1.LineItems))
	}
	for i, want := range feeAndWaiver {
		if got1.LineItems[i].Description != want.Description {
			t.Errorf("#1 line %d = %q, want %q (lines must come back in the order sent)", i+1, got1.LineItems[i].Description, want.Description)
		}
	}
	var sum int64
	var sawWaiver bool
	for _, l := range got1.LineItems {
		sum += l.Amount
		if l.Amount == -700_00 {
			sawWaiver = true
		}
	}
	if sum != got1.Total || !sawWaiver {
		t.Errorf("#1 Σ lines %d vs total %d, waiver line present=%v", sum, got1.Total, sawWaiver)
	}
	if !got1.CreatedAt.Equal(aug1) {
		t.Errorf("#1 invoice date = %v, want 2026-08-01", got1.CreatedAt)
	}
	if want := aug1.AddDate(0, 0, 30); !got1.DueDate.Equal(want) {
		t.Errorf("#1 due = %v, want %v", got1.DueDate, want)
	}
	if got1.EntityID == nil || *got1.EntityID != duk.ID {
		t.Errorf("#1 entity = %v, want UKCO %s", got1.EntityID, duk.ID)
	}
	if got1.DunningManagedBy != "manual" {
		t.Errorf("#1 dunning_managed_by = %q, want manual", got1.DunningManagedBy)
	}

	// Ledger: exactly one Code-1 leg == the NET total, on the UKCO entity's ledger.
	var legCount int
	var legSum int64
	var legLedger int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(amount),0), COALESCE(MAX(ledger_id),0) FROM ledger_transactions WHERE reference_id=$1 AND code=1`,
		inv1.ID).Scan(&legCount, &legSum, &legLedger); err != nil {
		t.Fatalf("read ledger leg: %v", err)
	}
	if legCount != 1 || legSum != got1.Total || legLedger != duk.TBLedgerID {
		t.Errorf("ledger leg count=%d sum=%d ledger=%d, want 1 / %d / %d", legCount, legSum, legLedger, got1.Total, duk.TBLedgerID)
	}

	// 2. Replaying the key returns the original, creates nothing, burns no number.
	again, created, err := svc.CreateManualInvoice(ctx, newIn(duk, "cust-a|UKCO|2026-08", feeAndWaiver))
	if err != nil || created || again.ID != inv1.ID || again.InvoiceNumber != "UKCO-000001" {
		t.Fatalf("replay: created=%v id-match=%v number=%q err=%v", created, again != nil && again.ID == inv1.ID, again.InvoiceNumber, err)
	}
	// The same key with DIFFERENT content must not be silently replayed.
	changed := feeAndWaiver[:2:2] // drop the waiver → different total and line count
	if _, _, err := svc.CreateManualInvoice(ctx, newIn(duk, "cust-a|UKCO|2026-08", changed)); !errors.Is(err, ErrManualInvoiceRefConflict) {
		t.Errorf("same key, different lines: err = %v, want ErrManualInvoiceRefConflict", err)
	}
	if _, _, err := svc.CreateManualInvoice(ctx, newIn(dsg, "cust-a|UKCO|2026-08", feeAndWaiver)); !errors.Is(err, ErrManualInvoiceRefConflict) {
		t.Errorf("same key, different entity: err = %v, want ErrManualInvoiceRefConflict", err)
	}
	inv2, created, err := svc.CreateManualInvoice(ctx, newIn(duk, "cust-b|UKCO|2026-08",
		lines(ManualInvoiceLine{Description: "Platform fee", Amount: 2500_00})))
	if err != nil || !created || inv2.InvoiceNumber != "UKCO-000002" {
		t.Fatalf("next invoice after replay: number=%q created=%v err=%v, want UKCO-000002 (no gap)", inv2.InvoiceNumber, created, err)
	}

	// 3. A different entity has its own independent series.
	inv3, _, err := svc.CreateManualInvoice(ctx, newIn(dsg, "cust-a|SGCO|2026-08",
		lines(ManualInvoiceLine{Description: "Support fee", Amount: 99_00})))
	if err != nil || inv3.InvoiceNumber != "SGCO-000001" {
		t.Fatalf("SGCO invoice: number=%q err=%v, want SGCO-000001", inv3.InvoiceNumber, err)
	}

	// 4. Fully-waived invoice → paid, $0, reconciles, never an overdue candidate.
	zero, _, err := svc.CreateManualInvoice(ctx, newIn(duk, "cust-a|UKCO|2026-08-zero", lines(
		ManualInvoiceLine{Description: "Utility fee", Amount: 4002_18},
		ManualInvoiceLine{Description: "Utility fee waiver", Amount: -4002_18},
	)))
	if err != nil {
		t.Fatalf("net-zero invoice: %v", err)
	}
	gotZero, err := invRepo.GetByID(context.WithValue(ctx, domain.TenantIDKey, tenantID), zero.ID)
	if err != nil || gotZero.Total != 0 || gotZero.Status != domain.InvoiceStatusPaid || gotZero.AmountPaid != 0 {
		t.Fatalf("net-zero reload: err=%v total=%d status=%q paid=%d", err, gotZero.Total, gotZero.Status, gotZero.AmountPaid)
	}

	// 5. The reconciler's own queries find nothing wrong with any of them.
	ledgerRepo := db.NewLedgerRepository(conn)
	if rows, n, err := ledgerRepo.GetInvoiceLedgerMismatches(ctx, tenantID, 50); err != nil || n != 0 {
		t.Errorf("invoice↔ledger mismatches: n=%d rows=%v err=%v, want none", n, rows, err)
	}
	if rows, n, err := ledgerRepo.GetPaymentLedgerMismatches(ctx, tenantID, 50); err != nil || n != 0 {
		t.Errorf("payment↔ledger mismatches: n=%d rows=%v err=%v, want none", n, rows, err)
	}
	if lines, err := ledgerRepo.GetTrialBalanceLines(ctx, tenantID, nil); err != nil {
		t.Errorf("trial balance: %v", err)
	} else {
		var debits, credits int64
		for _, l := range lines {
			debits += l.Debits
			credits += l.Credits
		}
		if debits != credits {
			t.Errorf("trial balance out of balance: debits %d != credits %d", debits, credits)
		}
	}

	// 6. Imported invoices are past due (Aug 2026 vs. today) but the dunning
	// scheduler must not see them.
	overdue, err := invRepo.GetOverdueInvoices(ctx)
	if err != nil {
		t.Fatalf("overdue scan: %v", err)
	}
	for _, o := range overdue {
		if o.ID == inv1.ID || o.ID == inv2.ID || o.ID == inv3.ID || o.ID == zero.ID {
			t.Errorf("manual invoice %s is a dunning candidate; it must be excluded", o.InvoiceNumber)
		}
	}

	// 7. Concurrent replays of one key produce exactly one invoice and no gap.
	const racers = 8
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, racers)
	createdFlags := make([]bool, racers)
	errs := make([]error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			inv, c, e := svc.CreateManualInvoice(ctx, newIn(duk, "race|UKCO|2026-08",
				lines(ManualInvoiceLine{Description: "Race", Amount: 10_00})))
			errs[i], createdFlags[i] = e, c
			if inv != nil {
				ids[i] = inv.ID
			}
		}(i)
	}
	wg.Wait()
	winners := 0
	for i := 0; i < racers; i++ {
		if errs[i] != nil {
			t.Errorf("racer %d: %v", i, errs[i])
		}
		if createdFlags[i] {
			winners++
		}
		if ids[i] != ids[0] {
			t.Errorf("racer %d returned a different invoice (%s vs %s)", i, ids[i], ids[0])
		}
	}
	if winners != 1 {
		t.Errorf("%d racers reported created=true, want exactly 1", winners)
	}
	var rows int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM invoices WHERE tenant_id=$1 AND external_ref='race|UKCO|2026-08'`, tenantID).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("rows for raced key = %d err=%v, want 1", rows, err)
	}
	next, _, err := svc.CreateManualInvoice(ctx, newIn(duk, "after-race|UKCO|2026-08",
		lines(ManualInvoiceLine{Description: "After race", Amount: 5_00})))
	if err != nil {
		t.Fatalf("post-race invoice: %v", err)
	}
	// UKCO so far: 000001, 000002, zero(000003), race winner(000004) → next is 000005.
	if next.InvoiceNumber != "UKCO-000005" {
		t.Errorf("post-race number = %q, want UKCO-000005 (losing racers must not burn numbers)", next.InvoiceNumber)
	}

	// 8. Without a key nothing dedupes; with bad references the caller gets a 400-class error.
	n1, _, e1 := svc.CreateManualInvoice(ctx, newIn(duk, "", lines(ManualInvoiceLine{Description: "d", Amount: 1_00})))
	n2, _, e2 := svc.CreateManualInvoice(ctx, newIn(duk, "", lines(ManualInvoiceLine{Description: "d", Amount: 1_00})))
	if e1 != nil || e2 != nil || n1.ID == n2.ID {
		t.Errorf("keyless calls must create separate invoices: %v %v", e1, e2)
	}
	badEntity := newIn(nil, "", lines(ManualInvoiceLine{Description: "d", Amount: 1_00}))
	unknown := uuid.New()
	badEntity.EntityID = &unknown
	if _, _, err := svc.CreateManualInvoice(ctx, badEntity); !errors.Is(err, ErrInvalidManualInvoice) {
		t.Errorf("unknown entity: err = %v, want ErrInvalidManualInvoice", err)
	}
	foreign := newIn(duk, "", lines(ManualInvoiceLine{Description: "d", Amount: 1_00}))
	foreign.CustomerID = foreignCust
	if _, _, err := svc.CreateManualInvoice(ctx, foreign); !errors.Is(err, ErrInvalidManualInvoice) {
		t.Errorf("other tenant's customer: err = %v, want ErrInvalidManualInvoice (tenant isolation)", err)
	}
	// The same key in another tenant is independent.
	otherCust := seedCustomer(otherTenant, "Other")
	oi, oc, err := svc.CreateManualInvoice(ctx, CreateManualInvoiceInput{
		TenantID: otherTenant, CustomerID: otherCust, Currency: "USD", InvoiceDate: aug1,
		ExternalRef: "cust-a|UKCO|2026-08", Lines: lines(ManualInvoiceLine{Description: "d", Amount: 1_00}),
	})
	if err != nil || !oc || oi.ID == inv1.ID {
		t.Errorf("same key in another tenant: created=%v err=%v, want an independent invoice", oc, err)
	}
}
