package handler

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
)

// TestCancelSubscriptionRequest_ReasonValidation locks in the binding rules on
// the cancellation reason: it is required and must be one of the known
// CancellationReason values, so an absent or unrecognized reason is rejected at
// the boundary instead of reaching the service as free-form text.
func TestCancelSubscriptionRequest_ReasonValidation(t *testing.T) {
	t.Run("known reason passes", func(t *testing.T) {
		req := CancelSubscriptionRequest{Reason: ReasonSwitching}
		if err := binding.Validator.ValidateStruct(&req); err != nil {
			t.Errorf("valid reason %q rejected: %v", req.Reason, err)
		}
	})

	t.Run("unknown reason fails", func(t *testing.T) {
		req := CancelSubscriptionRequest{Reason: CancellationReason("bogus_reason")}
		if err := binding.Validator.ValidateStruct(&req); err == nil {
			t.Error("unrecognized reason accepted; oneof validation not enforced")
		}
	})

	t.Run("empty reason fails", func(t *testing.T) {
		req := CancelSubscriptionRequest{}
		if err := binding.Validator.ValidateStruct(&req); err == nil {
			t.Error("empty reason accepted; required validation not enforced")
		}
	})
}
