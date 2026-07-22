package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListTools_IncludesTier2Writes(t *testing.T) {
	cs := connectInMemory(t, NewServer(NewClient("http://unused"), Options{}))
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*mcp.Tool{}
	for _, tl := range res.Tools {
		got[tl.Name] = tl
	}
	// Tier-1 (14) + Tier-2 (9) both on by default.
	if len(got) != len(readToolPaths)+len(writeToolPaths) {
		t.Fatalf("registered %d tools, want %d", len(got), len(readToolPaths)+len(writeToolPaths))
	}
	for name := range writeToolPaths {
		tl, ok := got[name]
		if !ok {
			t.Errorf("write tool %q not registered", name)
			continue
		}
		if tl.Annotations == nil || !tl.Annotations.IdempotentHint {
			t.Errorf("Tier-2 tool %q must carry IdempotentHint", name)
		}
		if tl.Annotations.ReadOnlyHint {
			t.Errorf("write tool %q must not be marked read-only", name)
		}
	}
}

func TestTierGate_DisablingTier2HidesWritesButKeepsReads(t *testing.T) {
	cs := connectInMemory(t, NewServer(NewClient("http://unused"),
		Options{AllowTiers: tierSet{Tier1: true, Tier2: false}}))
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range res.Tools {
		if _, isWrite := writeToolPaths[tl.Name]; isWrite {
			t.Errorf("Tier-2 disabled but write tool %q is exposed", tl.Name)
		}
	}
	if len(res.Tools) != len(readToolPaths) {
		t.Errorf("expected %d read tools, got %d", len(readToolPaths), len(res.Tools))
	}
}

func TestWrite_SendsIdempotencyKeyAndForwardsCallerKey(t *testing.T) {
	var mu sync.Mutex
	var idemSeen, authSeen string
	var body map[string]any
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		idemSeen = r.Header.Get("Idempotency-Key")
		authSeen = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"data":{"id":"cust_new"}}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "rsk_test_CALLER")
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_customer",
		Arguments: map[string]any{"email": "a@b.com", "name": "ACME", "idempotency_key": "stable-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", resultText(res))
	}
	if idemSeen != "stable-1" {
		t.Errorf("Idempotency-Key = %q, want the caller's stable key", idemSeen)
	}
	if authSeen != "Bearer rsk_test_CALLER" {
		t.Errorf("Authorization = %q", authSeen)
	}
	if body["email"] != "a@b.com" || body["name"] != "ACME" {
		t.Errorf("body = %v", body)
	}
	if _, leaked := body["idempotency_key"]; leaked {
		t.Error("idempotency_key must be a header, not part of the /v1 body")
	}
}

func TestWrite_AutogeneratesIdempotencyKeyWhenOmitted(t *testing.T) {
	var got string
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		_, _ = io.WriteString(w, `{"data":{}}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "rsk_test_x")
	_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_customer",
		Arguments: map[string]any{"email": "a@b.com", "name": "ACME"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("a write with no idempotency_key must still send a generated Idempotency-Key")
	}
}

func TestWrite_ValidationFailsBeforeHittingV1(t *testing.T) {
	var calls int
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "rsk_test_x")
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_customer",
		Arguments: map[string]any{"email": ""}, // missing name + email
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected a validation error")
	}
	if calls != 0 {
		t.Errorf("/v1 must not be called on invalid input; got %d call(s)", calls)
	}
}

// TestStdioMode_UsesStaticKey verifies the local single-tenant mode: with a
// static key set, calls succeed even without an Authorization header (in-memory
// transport carries none) and the static key reaches /v1.
func TestStdioMode_UsesStaticKey(t *testing.T) {
	var authSeen string
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authSeen = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"data":[]}`)
	}))
	defer v1.Close()

	srv := NewServer(NewClient(v1.URL), Options{StaticKey: "rsk_test_ENVKEY"})
	cs := connectInMemory(t, srv) // in-memory transport has no HTTP headers
	defer func() { _ = cs.Close() }()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_customers",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("stdio-mode call failed: %s", resultText(res))
	}
	if authSeen != "Bearer rsk_test_ENVKEY" {
		t.Errorf("/v1 saw %q, want the static env key", authSeen)
	}
}
