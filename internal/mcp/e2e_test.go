package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// authRoundTripper injects a bearer key on every outbound request, standing in
// for how a real MCP client authenticates to the Recurso MCP server.
type authRoundTripper struct {
	key  string
	base http.RoundTripper
}

func (a authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if a.key != "" {
		r.Header.Set("Authorization", "Bearer "+a.key)
	}
	return a.base.RoundTrip(r)
}

// connectHTTP fronts srv with the Streamable HTTP handler and connects an MCP
// client whose requests carry the given key (empty = unauthenticated).
func connectHTTP(t *testing.T, srv *Server, key string) *mcp.ClientSession {
	t.Helper()
	front := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv.MCP()
	}, nil))
	t.Cleanup(front.Close)

	httpClient := &http.Client{Transport: authRoundTripper{key: key, base: http.DefaultTransport}}
	tr := &mcp.StreamableClientTransport{Endpoint: front.URL, HTTPClient: httpClient}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(context.Background(), tr, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestEndToEnd_ForwardsCallerKeyToV1(t *testing.T) {
	var seenKey string
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/customers" {
			t.Errorf("unexpected /v1 path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"cust_1"}]}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "rsk_test_CALLER")

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_customers",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool reported error: %s", resultText(res))
	}
	if seenKey != "Bearer rsk_test_CALLER" {
		t.Errorf("/v1 saw %q — the caller's key must be forwarded verbatim", seenKey)
	}
	if !strings.Contains(resultText(res), "cust_1") {
		t.Errorf("result = %s", resultText(res))
	}
}

func TestEndToEnd_MissingKeyFailsClosed(t *testing.T) {
	var calls int
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "") // no key

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_customers",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected a tool error when no API key is supplied")
	}
	if calls != 0 {
		t.Errorf("/v1 must never be called without a key; saw %d call(s)", calls)
	}
}

func TestEndToEnd_SurfacesV1ErrorMessage(t *testing.T) {
	v1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"message":"subscription not found"}}`)
	}))
	defer v1.Close()

	cs := connectHTTP(t, NewServer(NewClient(v1.URL), Options{}), "rsk_test_x")

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_subscription",
		Arguments: map[string]any{"id": "sub_missing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(resultText(res), "subscription not found") {
		t.Errorf("expected surfaced /v1 message, got IsError=%v %q", res.IsError, resultText(res))
	}
}
