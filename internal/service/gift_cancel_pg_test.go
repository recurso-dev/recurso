package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCancelGift_PaidPath_Postgres is the money-path oracle for gift
// cancellation (policy: account credit): canceling a PAID, unredeemed gift
// issues the buyer a spendable adjustment credit note for exactly the amount
// paid, posts the GL issuance leg, and a second cancel is refused — the credit
// can never issue twice.
func TestCancelGift_PaidPath_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed gift-cancel test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	run := uuid.New().String()[:8]

	tenantID := seedRevRecTenant(t, conn)
	// The gift handler injects the tenant into the request context (the
	// customer repo's GetByID requires it) — mirror that here.
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
	buyerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Buyer', $4, NOW(), NOW())`,
		buyerID, tenantID, "buyer-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed buyer: %v", err)
	}
	planID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'Gifted',$3,'month',1,TRUE)`,
		planID, tenantID, "gift-"+run); err != nil {
		t.Fatalf("seed plan: %v", err)
	}

	// A PAID gift purchase invoice + the linked gift.
	const paid = int64(29900)
	invID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, billing_reason, created_at, due_date, paid_at)
		 VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,'gift_purchase',NOW(),NOW(),NOW())`,
		invID, tenantID, buyerID, paid, "INV-GIFT-"+run); err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	giftID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO gifts (id, tenant_id, code, plan_id, buyer_customer_id, recipient_email, status, duration_months, invoice_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'friend@x.com','purchased',6,$6,NOW(),NOW())`,
		giftID, tenantID, "GIFT-"+run, planID, buyerID, invID); err != nil {
		t.Fatalf("seed gift: %v", err)
	}

	// Real repos + real credit-note service with the ledger wired (main.go shape).
	sqlxDB := sqlx.NewDb(conn, "postgres")
	giftRepo := db.NewGiftRepository(sqlxDB)
	invoiceRepo := db.NewInvoiceRepository(conn)
	customerRepo := db.NewCustomerRepository(sqlxDB)
	creditNoteRepo := db.NewCreditNoteRepository(sqlxDB)
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	cn := NewCreditNoteService(creditNoteRepo, customerRepo, invoiceRepo, nil)
	cn.SetLedgerService(ledger)

	svc := NewGiftService(giftRepo, nil, &InvoiceService{InvoiceRepo: invoiceRepo}, nil, nil)
	svc.SetCreditNoteService(cn)

	res, err := svc.CancelGift(ctx, tenantID, giftID, uuid.New(), "admin")
	if err != nil {
		t.Fatalf("CancelGift: %v", err)
	}
	if res.CreditNote == nil {
		t.Fatal("paid gift cancel must issue a credit note")
	}
	if res.CreditNote.Amount != paid || res.CreditNote.CustomerID != buyerID {
		t.Errorf("credit note = amount %d customer %s, want %d / buyer", res.CreditNote.Amount, res.CreditNote.CustomerID, paid)
	}
	if res.InvoiceVoided {
		t.Error("a paid invoice must not be voided")
	}

	// The credit is spendable (issued, full balance) and the GL issuance leg posted.
	var status string
	var balance int64
	if err := conn.QueryRowContext(ctx,
		`SELECT status, balance FROM credit_notes WHERE id=$1`, res.CreditNote.ID).Scan(&status, &balance); err != nil {
		t.Fatalf("read credit note: %v", err)
	}
	if status != "issued" || balance != paid {
		t.Errorf("credit note %s/%d, want issued/%d (spendable)", status, balance, paid)
	}
	var legs int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id=$1`, res.CreditNote.ID).Scan(&legs); err != nil {
		t.Fatalf("count GL legs: %v", err)
	}
	if legs == 0 {
		t.Error("credit issuance posted no GL leg — books diverge from the spendable balance")
	}

	// Second cancel: refused, and still exactly one credit note.
	if _, err := svc.CancelGift(ctx, tenantID, giftID, uuid.New(), "admin"); !errors.Is(err, ErrGiftAlreadyCanceled) {
		t.Fatalf("second cancel: got %v, want ErrGiftAlreadyCanceled", err)
	}
	var cnCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_notes WHERE customer_id=$1`, buyerID).Scan(&cnCount); err != nil {
		t.Fatalf("count credit notes: %v", err)
	}
	if cnCount != 1 {
		t.Fatalf("buyer has %d credit notes, want exactly 1 — cancel must never double-credit", cnCount)
	}

	// The gift is canceled and its code can no longer be redeemed.
	g, err := giftRepo.GetByID(ctx, tenantID, giftID)
	if err != nil || g == nil || g.Status != domain.GiftStatusCanceled {
		t.Fatalf("gift after cancel = %+v (err %v), want canceled", g, err)
	}
	won, err := giftRepo.MarkRedeemed(ctx, giftID, tenantID, uuid.New(), g.UpdatedAt)
	if err != nil || won {
		t.Fatalf("a canceled gift must not be redeemable: won=%v err=%v", won, err)
	}
}
