package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type fakeConnOpener struct {
	conn   *domain.GatewayConnection
	getErr error
	secret string
	secErr error
}

func (f *fakeConnOpener) GetByID(_ context.Context, _ uuid.UUID) (*domain.GatewayConnection, error) {
	return f.conn, f.getErr
}
func (f *fakeConnOpener) OpenSecret(_ *domain.GatewayConnection) (string, error) {
	return f.secret, f.secErr
}

// namedCharger records which gateway a charge was routed to.
type namedCharger struct{ name string }

func (c *namedCharger) ChargeSavedPaymentMethod(_ context.Context, _, _ string, _ int64, _, _, _ string) (*port.PaymentResult, error) {
	return &port.PaymentResult{Success: true}, nil
}

func TestSavedCardGatewayRouter_ChargerFor(t *testing.T) {
	platform := &namedCharger{name: "platform"}
	build := func(secret string) SavedCardCharger { return &namedCharger{name: "byo:" + secret} }

	t.Run("nil connection -> platform", func(t *testing.T) {
		r := NewSavedCardGatewayRouter(&fakeConnOpener{}, build, platform)
		got, err := r.ChargerFor(context.Background(), nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != SavedCardCharger(platform) {
			t.Fatalf("nil connection should route to platform, got %#v", got)
		}
	})

	t.Run("stripe connection -> byo built from secret", func(t *testing.T) {
		connID := uuid.New()
		opener := &fakeConnOpener{
			conn:   &domain.GatewayConnection{ID: connID, Provider: domain.GatewayStripe, SecretKeyEnc: "sealed"},
			secret: "sk_byo",
		}
		r := NewSavedCardGatewayRouter(opener, build, platform)
		got, err := r.ChargerFor(context.Background(), &connID)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		nc, ok := got.(*namedCharger)
		if !ok || nc.name != "byo:sk_byo" {
			t.Fatalf("stripe connection should build BYO charger from its secret, got %#v", got)
		}
	})

	t.Run("connection load error -> error (dunning)", func(t *testing.T) {
		connID := uuid.New()
		r := NewSavedCardGatewayRouter(&fakeConnOpener{getErr: errors.New("db down")}, build, platform)
		if _, err := r.ChargerFor(context.Background(), &connID); err == nil {
			t.Fatal("expected error when the connection can't be loaded")
		}
	})

	t.Run("missing connection -> error", func(t *testing.T) {
		connID := uuid.New()
		r := NewSavedCardGatewayRouter(&fakeConnOpener{conn: nil}, build, platform)
		if _, err := r.ChargerFor(context.Background(), &connID); err == nil {
			t.Fatal("expected error when the connection is gone")
		}
	})

	t.Run("non-stripe connection -> error", func(t *testing.T) {
		connID := uuid.New()
		opener := &fakeConnOpener{conn: &domain.GatewayConnection{ID: connID, Provider: domain.GatewayRazorpay, SecretKeyEnc: "sealed"}, secret: "s"}
		r := NewSavedCardGatewayRouter(opener, build, platform)
		if _, err := r.ChargerFor(context.Background(), &connID); err == nil {
			t.Fatal("expected error for a non-stripe saved-card connection")
		}
	})

	t.Run("secret open failure -> error", func(t *testing.T) {
		connID := uuid.New()
		opener := &fakeConnOpener{conn: &domain.GatewayConnection{ID: connID, Provider: domain.GatewayStripe, SecretKeyEnc: "sealed"}, secErr: errors.New("vault down")}
		r := NewSavedCardGatewayRouter(opener, build, platform)
		if _, err := r.ChargerFor(context.Background(), &connID); err == nil {
			t.Fatal("expected error when the secret can't be opened")
		}
	})
}
