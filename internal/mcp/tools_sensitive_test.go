package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSensitive_NotRegisteredByDefault(t *testing.T) {
	// Default policy has Tier-3 OFF, so the money-path tools must not appear at all.
	cs := connectInMemory(t, NewServer(NewClient("http://unused"), Options{}))
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range res.Tools {
		if _, isSensitive := sensitiveToolPaths[tl.Name]; isSensitive {
			t.Errorf("Tier-3 tool %q must not be registered under the default policy", tl.Name)
		}
	}
}

func TestSensitive_RegisteredWhenTier3Allowed(t *testing.T) {
	srv := NewServer(NewClient("http://unused"),
		Options{AllowTiers: tierSet{Tier1: true, Tier2: true, Tier3: true}})
	cs := connectInMemory(t, srv)
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*mcp.Tool{}
	for _, tl := range res.Tools {
		got[tl.Name] = tl
	}
	for name := range sensitiveToolPaths {
		tl, ok := got[name]
		if !ok {
			t.Errorf("Tier-3 tool %q not registered when Tier3 allowed", name)
			continue
		}
		if tl.Annotations == nil || tl.Annotations.DestructiveHint == nil || !*tl.Annotations.DestructiveHint {
			t.Errorf("Tier-3 tool %q must carry a destructive hint", name)
		}
	}
}

// TestSensitive_RefusesWhenTenantNotOptedIn is the safety-critical test: with
// Tier-3 in the policy but the TENANT opt-in off, a money-path tool must refuse
// and MUST NOT call its /v1 money-path endpoint.
func TestSensitive_RefusesWhenTenantNotOptedIn(t *testing.T) {
	var moneyPathCalls int32
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/settings/mcp" {
			_, _ = io.WriteString(w, `{"data":{"tier3_enabled":false}}`) // NOT opted in
			return
		}
		atomic.AddInt32(&moneyPathCalls, 1) // any other path = the money-path action
		_, _ = io.WriteString(w, `{"data":{}}`)
	}))
	defer v1.Close()

	srv := NewServer(NewClient(v1.URL), Options{AllowTiers: tierSet{Tier1: true, Tier3: true}})
	cs := connectHTTP(t, srv, "rsk_test_x")

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "cancel_subscription",
		Arguments: map[string]any{"id": "sub_1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("Tier-3 tool must refuse when the tenant has not opted in")
	}
	if !strings.Contains(resultText(res), "disabled") {
		t.Errorf("expected an opt-in-required message, got %q", resultText(res))
	}
	if n := atomic.LoadInt32(&moneyPathCalls); n != 0 {
		t.Errorf("money-path endpoint must NOT be called when opt-in is off; got %d call(s)", n)
	}
}

func TestSensitive_ProceedsWhenOptedIn(t *testing.T) {
	var idemSeen string
	var canceled int32
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/settings/mcp" {
			_, _ = io.WriteString(w, `{"data":{"tier3_enabled":true}}`) // opted in
			return
		}
		if r.URL.Path == "/v1/subscriptions/sub_1/cancel" {
			atomic.AddInt32(&canceled, 1)
			idemSeen = r.Header.Get("Idempotency-Key")
		}
		_, _ = io.WriteString(w, `{"data":{"status":"canceled"}}`)
	}))
	defer v1.Close()

	srv := NewServer(NewClient(v1.URL), Options{AllowTiers: tierSet{Tier1: true, Tier3: true}})
	cs := connectHTTP(t, srv, "rsk_test_x")

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "cancel_subscription",
		Arguments: map[string]any{"id": "sub_1", "idempotency_key": "cancel-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", resultText(res))
	}
	if atomic.LoadInt32(&canceled) != 1 {
		t.Error("cancel endpoint should have been called once")
	}
	if idemSeen != "cancel-1" {
		t.Errorf("Idempotency-Key = %q, want the caller's stable key", idemSeen)
	}
}

// TestSensitive_FailClosedOnSettingsError verifies fail-closed behavior: if the
// opt-in check itself errors, the tool refuses rather than proceeding.
func TestSensitive_FailClosedOnSettingsError(t *testing.T) {
	var moneyPathCalls int32
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/settings/mcp" {
			w.WriteHeader(http.StatusInternalServerError) // opt-in check fails
			return
		}
		atomic.AddInt32(&moneyPathCalls, 1)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer v1.Close()

	srv := NewServer(NewClient(v1.URL), Options{AllowTiers: tierSet{Tier1: true, Tier3: true}})
	cs := connectHTTP(t, srv, "rsk_test_x")

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "wallet_top_up",
		Arguments: map[string]any{"id": "w1", "amount": 500},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("must fail closed when the opt-in check errors")
	}
	if n := atomic.LoadInt32(&moneyPathCalls); n != 0 {
		t.Errorf("money-path must not run when opt-in is indeterminate; got %d call(s)", n)
	}
}
