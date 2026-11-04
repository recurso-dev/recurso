package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
)

func TestCheckStripeKey(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		body        string
		wantAccount string
		wantErr     string
	}{
		{"a working key", http.StatusOK, `{"id":"acct_1","object":"account"}`, "acct_1", ""},
		{"a restricted key without account access is still valid", http.StatusForbidden, `{"error":{"type":"invalid_request_error","message":"no"}}`, "", ""},
		{"a rejected key", http.StatusUnauthorized, `{"error":{"type":"invalid_request_error","message":"Invalid API Key"}}`, "", "rejected"},
		{"a Stripe outage isn't blamed on the key", http.StatusInternalServerError, `{"error":{"type":"api_error","message":"down"}}`, "", "try again"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			zero := int64(0)
			backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
				URL: stripe.String(srv.URL), MaxNetworkRetries: &zero, LeveledLogger: &stripe.LeveledLogger{Level: stripe.LevelNull},
			})
			account, err := checkStripeKey(client.New("sk_test_x", &stripe.Backends{API: backend, Connect: backend, Uploads: backend}))
			if account != tc.wantAccount {
				t.Errorf("account = %q, want %q", account, tc.wantAccount)
			}
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v, want none", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestCheckStripeKey_UnreachableStripe(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	zero := int64(0)
	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL: stripe.String(url), MaxNetworkRetries: &zero, LeveledLogger: &stripe.LeveledLogger{Level: stripe.LevelNull},
	})
	_, err := checkStripeKey(client.New("sk_test_x", &stripe.Backends{API: backend, Connect: backend, Uploads: backend}))
	if err == nil || !strings.Contains(err.Error(), "could not reach") {
		t.Errorf("err = %v, want a could-not-reach error", err)
	}
}
