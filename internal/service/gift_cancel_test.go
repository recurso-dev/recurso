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
type giftCancelInvoiceRepo struct {
	port.InvoiceRepository
	inv        *domain.Invoice
	voidCalled int
}

func (r *giftCancelInvoiceRepo) GetByIDPublic(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}

func (r *giftCancelInvoiceRepo) VoidIfOpen(_ context.Context, _, _ uuid.UUID) (bool, error) {
	r.voidCalled++
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
