package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/stripe/stripe-go/v76"
)

// mockSubDeletedRepo overrides only GetByStripeSubscriptionID; the rest of the
// SubscriptionRepository surface is unused by handleSubscriptionDeleted.
type mockSubDeletedRepo struct {
	port.SubscriptionRepository
	sub *domain.Subscription
	err error
}

func (m *mockSubDeletedRepo) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	return m.sub, m.err
}

// A customer.subscription.deleted for a Stripe subscription with no local
// mapping must be ACKed (nil error → 200), not turned into a 500 — otherwise
// Stripe redelivers the deletion forever.
func TestHandleSubscriptionDeleted_UnknownSubscriptionIsAcked(t *testing.T) {
	h := &WebhookHandler{
		logger: slog.Default(),
		// Mirror the repository, which wraps the raw sql.ErrNoRows.
		subRepo: &mockSubDeletedRepo{err: fmt.Errorf("failed to get subscription by stripe ID: %w", sql.ErrNoRows)},
	}

	raw, _ := json.Marshal(stripe.Subscription{ID: "sub_stripe_unknown"})
	event := stripe.Event{
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{Raw: raw},
	}

	if err := h.handleSubscriptionDeleted(context.Background(), event); err != nil {
		t.Fatalf("unknown subscription must be acked (nil error), got %v", err)
	}
}

// A genuine infrastructure error (not a missing row) must still surface so the
// delivery is retried rather than silently dropped.
func TestHandleSubscriptionDeleted_RealErrorStillFails(t *testing.T) {
	h := &WebhookHandler{
		logger:  slog.Default(),
		subRepo: &mockSubDeletedRepo{err: fmt.Errorf("connection refused")},
	}

	raw, _ := json.Marshal(stripe.Subscription{ID: "sub_stripe_x"})
	event := stripe.Event{
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{Raw: raw},
	}

	if err := h.handleSubscriptionDeleted(context.Background(), event); err == nil {
		t.Fatal("a non-not-found lookup error must be surfaced (so Stripe retries), got nil")
	}
}
