package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectInMemory wires an MCP client to srv over an in-memory transport and
// returns the client session. Listing tools needs no auth, so this is enough
// for surface/gate assertions.
func connectInMemory(t *testing.T, srv *Server) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.MCP().Connect(ctx, st, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	return cs
}

// resultText concatenates the text content of a tool result.
func resultText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestListTools_Tier1SurfaceMatchesCatalogue(t *testing.T) {
	cs := connectInMemory(t, NewServer(NewClient("http://unused"), Options{}))
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*mcp.Tool{}
	readOnly := 0
	for _, tl := range res.Tools {
		got[tl.Name] = tl
		if tl.Annotations != nil && tl.Annotations.ReadOnlyHint {
			readOnly++
		}
	}
	if readOnly != len(readToolPaths) {
		t.Fatalf("registered %d read-only tools, catalogue has %d", readOnly, len(readToolPaths))
	}
	for name := range readToolPaths {
		tl, ok := got[name]
		if !ok {
			t.Errorf("tool %q registered? no", name)
			continue
		}
		if tl.Annotations == nil || !tl.Annotations.ReadOnlyHint {
			t.Errorf("Tier-1 tool %q must carry ReadOnlyHint", name)
		}
		if strings.TrimSpace(tl.Description) == "" {
			t.Errorf("tool %q has no description", name)
		}
	}
}

func TestTierGate_DisablingTier1RegistersNothing(t *testing.T) {
	cs := connectInMemory(t, NewServer(NewClient("http://unused"), Options{AllowTiers: tierSet{Tier1: false}}))
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != 0 {
		t.Errorf("Tier1 disabled → expected 0 tools, got %d", len(res.Tools))
	}
}
