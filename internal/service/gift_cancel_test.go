package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// giftCancelInvoiceRepo stubs the two invoice reads/writes CancelGift uses.
// payAfterReads simulates a checkout settling mid-cancel: after that many
// GetByIDPublic calls, the invoice flips open→paid.
type giftCancelInvoiceRepo struct {
	port.InvoiceRepository
	inv           *domain.Invoice
	voidCalled    int
	reads         int
	payAfterReads int
}

func (r *giftCancelInvoiceRepo) GetByIDPublic(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	r.reads++
	return r.inv, nil
}

// settleIfRaced flips open→paid once the pre-check read has happened — the
// simulated checkout settles between that read and the void.
func (r *giftCancelInvoiceRepo) settleIfRaced() {
	if r.payAfterReads > 0 && r.reads >= r.payAfterReads && r.inv != nil &&
		r.inv.Status == domain.InvoiceStatusOpen {
		r.inv.Status = domain.InvoiceStatusPaid
	}
}

func (r *giftCancelInvoiceRepo) VoidIfOpen(_ context.Context, _, _ uuid.UUID) (bool, error) {
	r.voidCalled++
	r.settleIfRaced()
	if r.inv != nil && r.inv.Status == domain.InvoiceStatusOpen {
		r.inv.Status = domain.InvoiceStatusVoid
		return true, nil
	}
	return false, nil
}

func seedCancelGift(repo *mockGiftRepo, status domain.GiftStatus, invoiceID *uuid.UUID) *domain.Gift {
	g := &domain.Gift{
		ID: uuid.New(), TenantID: uuid.New(), Code: "GIFT-CXL",
		BuyerCustomerID: uuid.New(), Status: status, InvoiceID: invoiceID,
	}
	repo.gifts[g.Code] = g
	return g
}

func TestCancelGift_Guards(t *testing.T) {
	repo := newMockGiftRepo()
	svc := NewGiftService(repo, nil, nil, nil, nil)

	// Unknown id → not found.
	if _, err := svc.CancelGift(context.Background(), uuid.New(), uuid.New(), uuid.New(), "admin"); !errors.Is(err, ErrGiftNotFound) {
		t.Errorf("unknown gift: got %v, want ErrGiftNotFound", err)
	}

	// Redeemed → refused; the recipient's subscription is not clawed back.
	g := seedCancelGift(repo, domain.GiftStatusRedeemed, nil)
	if _, err := svc.CancelGift(context.Background(), g.TenantID, g.ID, uuid.New(), "admin"); !errors.Is(err, ErrGiftAlreadyRedeemed) {
		t.Errorf("redeemed gift: got %v, want ErrGiftAlreadyRedeemed", err)
	}

	// Already canceled → refused, so the credit can never issue twice.
	g2 := seedCancelGift(repo, domain.GiftStatusCanceled, nil)
	if _, err := svc.CancelGift(context.Background(), g2.TenantID, g2.ID, uuid.New(), "admin"); !errors.Is(err, ErrGiftAlreadyCanceled) {
		t.Errorf("canceled gift: got %v, want ErrGiftAlreadyCanceled", err)
	}
}

// A still-open (unpaid) purchase invoice is VOIDED, never credited — no money
// arrived, so compensating would invent liability.
func TestCancelGift_OpenInvoiceVoided(t *testing.T) {
	repo := newMockGiftRepo()
	invID := uuid.New()
	g := seedCancelGift(repo, domain.GiftStatusPurchased, &invID)
	invRepo := &giftCancelInvoiceRepo{inv: &domain.Invoice{
		ID: invID, TenantID: g.TenantID, Status: domain.InvoiceStatusOpen, Total: 9900, Currency: "USD",
	}}
	svc := NewGiftService(repo, nil, &InvoiceService{InvoiceRepo: invRepo}, nil, nil)

	res, err := svc.CancelGift(context.Background(), g.TenantID, g.ID, uuid.New(), "admin")
	if err != nil {
		t.Fatalf("CancelGift: %v", err)
	}
	if !res.InvoiceVoided || res.CreditNote != nil {
		t.Errorf("open invoice: voided=%v credit=%v, want voided + no credit", res.InvoiceVoided, res.CreditNote)
	}
	if g.Status != domain.GiftStatusCanceled {
		t.Errorf("gift status = %s, want canceled", g.Status)
	}
	if invRepo.inv.Status != domain.InvoiceStatusVoid {
		t.Errorf("invoice status = %s, want void", invRepo.inv.Status)
	}
}

// A PAID purchase with no credit-note service wired is REFUSED before the
// status flips — never cancel-and-strand the buyer's money.
func TestCancelGift_PaidButUnwiredRefused(t *testing.T) {
	repo := newMockGiftRepo()
	invID := uuid.New()
	g := seedCancelGift(repo, domain.GiftStatusPurchased, &invID)
	invRepo := &giftCancelInvoiceRepo{inv: &domain.Invoice{
		ID: invID, TenantID: g.TenantID, Status: domain.InvoiceStatusPaid, Total: 9900, Currency: "USD",
	}}
	svc := NewGiftService(repo, nil, &InvoiceService{InvoiceRepo: invRepo}, nil, nil) // no SetCreditNoteService

	if _, err := svc.CancelGift(context.Background(), g.TenantID, g.ID, uuid.New(), "admin"); !errors.Is(err, ErrGiftCreditUnwired) {
		t.Fatalf("got %v, want ErrGiftCreditUnwired", err)
	}
	if g.Status != domain.GiftStatusPurchased {
		t.Errorf("gift status = %s — a refused cancel must not flip the status", g.Status)
	}
}

// A checkout payment can settle BETWEEN the pre-check read (invoice open) and
// the atomic cancel. The money side must follow the post-cancel truth: the
// void refuses (invoice now paid) and the buyer must be credited — or, when
// credit issuance is unwired, the cancel must surface a loud error rather than
// silently stranding the payment behind invoice_voided=false.
func TestCancelGift_PaymentRacesCancel(t *testing.T) {
	repo := newMockGiftRepo()
	invID := uuid.New()
	g := seedCancelGift(repo, domain.GiftStatusPurchased, &invID)
	invRepo := &giftCancelInvoiceRepo{
		inv: &domain.Invoice{
			ID: invID, TenantID: g.TenantID, Status: domain.InvoiceStatusOpen, Total: 9900, Currency: "USD",
		},
		payAfterReads: 1, // open at the pre-check, paid by the time the void runs
	}
	svc := NewGiftService(repo, nil, &InvoiceService{InvoiceRepo: invRepo}, nil, nil) // credit unwired

	_, err := svc.CancelGift(context.Background(), g.TenantID, g.ID, uuid.New(), "admin")
	if !errors.Is(err, ErrGiftCreditUnwired) {
		t.Fatalf("raced payment with unwired credit must surface loudly, got %v", err)
	}
	if invRepo.inv.Status != domain.InvoiceStatusPaid {
		t.Fatalf("test setup: invoice should have been paid mid-cancel, is %s", invRepo.inv.Status)
	}
	if g.Status != domain.GiftStatusCanceled {
		// The cancel itself won; only the compensation is outstanding — and the
		// returned error says exactly that.
		t.Errorf("gift status = %s, want canceled", g.Status)
	}
}

// A legacy gift with no linked invoice cancels status-only (logged; the
// operator compensates manually if warranted).
func TestCancelGift_NoLinkedInvoice(t *testing.T) {
	repo := newMockGiftRepo()
	g := seedCancelGift(repo, domain.GiftStatusPurchased, nil)
	svc := NewGiftService(repo, nil, nil, nil, nil)

	res, err := svc.CancelGift(context.Background(), g.TenantID, g.ID, uuid.New(), "admin")
	if err != nil {
		t.Fatalf("CancelGift: %v", err)
	}
	if res.CreditNote != nil || res.InvoiceVoided {
		t.Errorf("legacy gift must cancel status-only, got %+v", res)
	}
	if g.Status != domain.GiftStatusCanceled {
		t.Errorf("gift status = %s, want canceled", g.Status)
	}
}
