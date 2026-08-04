package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// These guards reject negative money amounts at the API edge so no downstream
// ledger/credit-note/billing path can ever be handed a negative charge (#20).
// Each guard runs before its service is used, so a nil service is safe here —
// the request never reaches it.

func TestDispute_NegativeCreditRejected(t *testing.T) {
	c, w := jsonCtx(http.MethodPost, "/x",
		`{"outcome":"accept","issue_credit":true,"credit_amount":-100}`)
	withTenant(c)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	NewDisputeHandler(nil).ResolveDispute(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("negative credit_amount: got %d, want 400", w.Code)
	}
}

func TestReferral_NegativeRewardRejected(t *testing.T) {
	c, w := jsonCtx(http.MethodPost, "/x",
		`{"referrer_id":"`+uuid.New().String()+`","referred_id":"`+uuid.New().String()+`","reward_amount":-500,"currency":"USD"}`)
	withTenant(c)
	NewReferralHandler(nil).CreateReferral(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("negative reward_amount: got %d, want 400", w.Code)
	}
}

func TestPlan_NegativeAmountRejected(t *testing.T) {
	c, w := jsonCtx(http.MethodPost, "/x",
		`{"name":"Neg","code":"neg","interval_unit":"month","interval_count":1,"amount":-1000,"currency":"USD"}`)
	withTenant(c)
	NewCatalogHandler(nil).CreatePlan(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("negative plan amount: got %d, want 400", w.Code)
	}
}

// toEvent is a pure conversion, so the usage guard is tested directly.
func TestUsage_NegativeDynamicAmountRejected(t *testing.T) {
	_, err := recordEventRequest{
		SubscriptionID: uuid.New().String(),
		CustomerID:     uuid.New().String(),
		Dimension:      "api_calls",
		Quantity:       1,
		DynamicAmount:  -50,
	}.toEvent()
	if err != errNegativeDynamicAmount {
		t.Fatalf("negative dynamic_amount: got %v, want errNegativeDynamicAmount", err)
	}

	// A non-negative amount still converts cleanly.
	if _, err := (recordEventRequest{
		SubscriptionID: uuid.New().String(),
		CustomerID:     uuid.New().String(),
		Dimension:      "api_calls",
		Quantity:       1,
		DynamicAmount:  50,
	}).toEvent(); err != nil {
		t.Fatalf("valid dynamic_amount rejected: %v", err)
	}
}
