package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/service"
)

// #637 oracle: the sentinel mapping — a missing customer/plan/subscription is
// a 404, an illegal state transition a 409; only genuine faults stay 500.
// Fails on the pre-#637 code, which returned 500 (create/cancel/update) or
// 400 validation_failed (pause/resume) for all of these.
func TestRespondSubscriptionErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{service.ErrSubscriptionNotFound, http.StatusNotFound, "not_found"},
		{service.ErrPlanNotFound, http.StatusNotFound, "not_found"},
		{service.ErrCustomerNotFound, http.StatusNotFound, "not_found"},
		{fmt.Errorf("%w: only active subscriptions can be paused", service.ErrInvalidSubscriptionState), http.StatusConflict, "conflict"},
		{fmt.Errorf("db exploded"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/subscriptions", nil)
		respondSubscriptionError(c, tc.err)
		if w.Code != tc.status {
			t.Fatalf("%v: status = %d, want %d", tc.err, w.Code, tc.status)
		}
		if !strings.Contains(w.Body.String(), tc.code) {
			t.Fatalf("%v: body %q missing code %q", tc.err, w.Body.String(), tc.code)
		}
	}
}
