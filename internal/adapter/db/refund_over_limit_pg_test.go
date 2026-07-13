package db

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCreateRefundWithinLimit_ConcurrentRefundsSerialize proves the refund
// double-issue fix: when several runners try to refund the same paid invoice's
// full amount at once, the invoice-row lock lets EXACTLY ONE through — the rest
// see the winner's pending note in the over-refund sum and are rejected. Before
// the fix, the plain SumActiveRefundsForInvoice + Create sequence let every
// runner read a stale zero total and each issue a gateway refund.
func TestCreateRefundWithinLimit_ConcurrentRefundsSerialize(t *testing.T) {
	dbx := openCreditAppTestDB(t) // skips unless TEST_DATABASE_URL is set
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewCreditNoteRepository(dbx)
	ctx := context.Background()

	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)
	invoiceID := seedInvoiceRow(t, conn, tenantID, customerID, 100000)
	const amountPaid = int64(100000)

	newNote := func(i int) *domain.CreditNote {
		now := time.Now().UTC()
		inv := invoiceID
		ref := fmt.Sprintf("CN-REFUND-%s-%d", invoiceID.String()[:8], i)
		return &domain.CreditNote{
			TenantID:     tenantID,
			CustomerID:   customerID,
			InvoiceID:    &inv,
			Reference:    &ref,
			Amount:       amountPaid, // each runner refunds the FULL paid amount
			Balance:      0,
			Currency:     "USD",
			Status:       domain.CreditNoteStatusIssued,
			Reason:       "refund concurrency test",
			Type:         domain.CreditNoteTypeRefund,
			RefundStatus: domain.RefundStatusPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	const runners = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	var mu sync.Mutex
	wins := 0

	for i := 0; i < runners; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			within, err := repo.CreateRefundWithinLimit(ctx, newNote(i), invoiceID, amountPaid)
			if err != nil {
				t.Errorf("CreateRefundWithinLimit: %v", err)
				return
			}
			if within {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("full-amount refund succeeded %d times, want exactly 1 (double-refund race)", wins)
	}
	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM credit_notes WHERE invoice_id = $1 AND type = $2`,
		invoiceID, domain.CreditNoteTypeRefund).Scan(&count); err != nil {
		t.Fatalf("count refund notes: %v", err)
	}
	if count != 1 {
		t.Fatalf("persisted %d refund notes, want exactly 1", count)
	}
}

// TestCreateRefundWithinLimit_PartialRefundsUpToPaid confirms the guard still
// allows legitimate partial refunds that sum to the paid amount, and rejects the
// one that would tip over.
func TestCreateRefundWithinLimit_PartialRefundsUpToPaid(t *testing.T) {
	dbx := openCreditAppTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewCreditNoteRepository(dbx)
	ctx := context.Background()

	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)
	invoiceID := seedInvoiceRow(t, conn, tenantID, customerID, 100000)
	const amountPaid = int64(100000)

	note := func(ref string, amount int64) *domain.CreditNote {
		now := time.Now().UTC()
		inv := invoiceID
		refStr := ref
		return &domain.CreditNote{
			TenantID: tenantID, CustomerID: customerID, InvoiceID: &inv,
			Reference: &refStr, Amount: amount, Balance: 0, Currency: "USD",
			Status: domain.CreditNoteStatusIssued, Reason: "partial",
			Type: domain.CreditNoteTypeRefund, RefundStatus: domain.RefundStatusPending,
			CreatedAt: now, UpdatedAt: now,
		}
	}

	within, err := repo.CreateRefundWithinLimit(ctx, note("CN-P1", 60000), invoiceID, amountPaid)
	if err != nil || !within {
		t.Fatalf("first partial refund 60000: within=%v err=%v, want within=true", within, err)
	}
	within, err = repo.CreateRefundWithinLimit(ctx, note("CN-P2", 40000), invoiceID, amountPaid)
	if err != nil || !within {
		t.Fatalf("second partial refund 40000 (sums to 100000): within=%v err=%v, want within=true", within, err)
	}
	// This one tips over the paid amount.
	within, err = repo.CreateRefundWithinLimit(ctx, note("CN-P3", 1), invoiceID, amountPaid)
	if err != nil {
		t.Fatalf("third refund err: %v", err)
	}
	if within {
		t.Fatal("refund exceeding the paid amount was allowed (over-refund)")
	}
}
