package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type fakeAutopayGateway struct {
	paymentMethodSetup
	card      *port.SavedCard
	cardErr   error
	prepared  []string
	prepErr   error
	createdID string
}

func (g *fakeAutopayGateway) EnsureStripeCustomer(_ context.Context, existingID, _, _ string) (string, error) {
	if existingID != "" {
		return existingID, nil
	}
	return g.createdID, nil
}

func (g *fakeAutopayGateway) PrepareOrderForAutopay(_ context.Context, orderID, stripeCustomerID string) error {
	g.prepared = append(g.prepared, orderID+"/"+stripeCustomerID)
	return g.prepErr
}

func (g *fakeAutopayGateway) SavedCardFromOrder(context.Context, string) (*port.SavedCard, error) {
	return g.card, g.cardErr
}

type fakeAutopayResolver struct {
	setup  paymentMethodSetup
	connID *uuid.UUID
}

func (r fakeAutopayResolver) SetupForTenant(context.Context, uuid.UUID) (paymentMethodSetup, *uuid.UUID, string) {
	return r.setup, r.connID, "pk"
}

type fakeAutopayStore struct {
	stripeID, methodID string
	savedStripeID      string
	savedMethod        string
	savedConn          *uuid.UUID
}

func (s *fakeAutopayStore) GetByIDPublic(_ context.Context, id uuid.UUID) (*domain.Customer, error) {
	return &domain.Customer{ID: id, Email: "c@example.com"}, nil
}
func (s *fakeAutopayStore) GetStripeCustomerID(context.Context, uuid.UUID) (string, error) {
	return s.stripeID, nil
}
func (s *fakeAutopayStore) SetStripeCustomerID(_ context.Context, _ uuid.UUID, id string) error {
	s.savedStripeID = id
	return nil
}
func (s *fakeAutopayStore) SetDefaultPaymentMethod(_ context.Context, _ uuid.UUID, methodID, _, _ string, _, _ int, conn *uuid.UUID) error {
	s.savedMethod = methodID
	s.savedConn = conn
	return nil
}
func (s *fakeAutopayStore) GetSavedPaymentMethod(context.Context, uuid.UUID) (string, string, *uuid.UUID, error) {
	return s.stripeID, s.methodID, nil, nil
}

func subscriptionInvoice() *domain.Invoice {
	subID := uuid.New()
	return &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), SubscriptionID: &subID}
}

func TestCheckoutAutopay_Prepare(t *testing.T) {
	oneOff := subscriptionInvoice()
	oneOff.SubscriptionID = nil

	for _, tc := range []struct {
		name         string
		inv          *domain.Invoice
		store        *fakeAutopayStore
		prepErr      error
		want         bool
		wantPrepared string
		wantStripeID string
	}{
		{"a new customer gets a stripe customer and a prepared order", subscriptionInvoice(), &fakeAutopayStore{}, nil, true, "pi_1/cus_new", "cus_new"},
		{"an existing stripe customer is reused", subscriptionInvoice(), &fakeAutopayStore{stripeID: "cus_old"}, nil, true, "pi_1/cus_old", ""},
		{"a saved method is left alone", subscriptionInvoice(), &fakeAutopayStore{stripeID: "cus_old", methodID: "pm_old"}, nil, false, "", ""},
		{"a one-off invoice has no renewals to save for", oneOff, &fakeAutopayStore{}, nil, false, "", ""},
		{"an order stripe won't update reports no saving", subscriptionInvoice(), &fakeAutopayStore{}, errors.New("already succeeded"), false, "pi_1/cus_new", "cus_new"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeAutopayGateway{createdID: "cus_new", prepErr: tc.prepErr}
			a := NewCheckoutAutopay(tc.store, fakeAutopayResolver{setup: gw})

			if got := a.Prepare(context.Background(), tc.inv, "pi_1"); got != tc.want {
				t.Errorf("Prepare = %v, want %v", got, tc.want)
			}
			prepared := ""
			if len(gw.prepared) > 0 {
				prepared = gw.prepared[0]
			}
			if prepared != tc.wantPrepared {
				t.Errorf("prepared %q, want %q", prepared, tc.wantPrepared)
			}
			if tc.store.savedStripeID != tc.wantStripeID {
				t.Errorf("saved stripe customer %q, want %q", tc.store.savedStripeID, tc.wantStripeID)
			}
		})
	}
}

func TestCheckoutAutopay_Save(t *testing.T) {
	connID := uuid.New()
	card := &port.SavedCard{CustomerID: "cus_1", PaymentMethodID: "pm_new", Brand: "visa", Last4: "4242"}

	for _, tc := range []struct {
		name  string
		store *fakeAutopayStore
		card  *port.SavedCard
		err   error
		want  string
	}{
		{"the paying card is saved", &fakeAutopayStore{stripeID: "cus_1"}, card, nil, "pm_new"},
		{"a method saved in the meantime is kept", &fakeAutopayStore{stripeID: "cus_1", methodID: "pm_old"}, card, nil, ""},
		{"another stripe customer's card is refused", &fakeAutopayStore{stripeID: "cus_2"}, card, nil, ""},
		{"an order that wasn't prepared saves nothing", &fakeAutopayStore{stripeID: "cus_1"}, nil, nil, ""},
		{"a gateway error saves nothing", &fakeAutopayStore{stripeID: "cus_1"}, nil, errors.New("down"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeAutopayGateway{card: tc.card, cardErr: tc.err}
			a := NewCheckoutAutopay(tc.store, fakeAutopayResolver{setup: gw, connID: &connID})

			a.Save(context.Background(), subscriptionInvoice(), "pi_1")

			if tc.store.savedMethod != tc.want {
				t.Errorf("saved %q, want %q", tc.store.savedMethod, tc.want)
			}
			if tc.want != "" && tc.store.savedConn != &connID {
				t.Errorf("connection = %v, want the tenant's so renewals charge the same account", tc.store.savedConn)
			}
		})
	}
}

// Handlers call these with autopay unwired, and gateways that can't save (the
// mock, Razorpay) resolve to nothing.
func TestCheckoutAutopay_NilSafe(t *testing.T) {
	var a *CheckoutAutopay
	inv := subscriptionInvoice()
	if a.Prepare(context.Background(), inv, "pi_1") {
		t.Error("nil Prepare reported saving")
	}
	a.Save(context.Background(), inv, "pi_1")

	unwired := NewCheckoutAutopay(&fakeAutopayStore{}, fakeAutopayResolver{})
	if unwired.Prepare(context.Background(), inv, "pi_1") {
		t.Error("Prepare reported saving with no capable gateway")
	}
	unwired.Save(context.Background(), inv, "pi_1")
}
