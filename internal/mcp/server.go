package mcp

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "recurso-billing"

// Server wraps the MCP server plus the /v1 client and the enabled-tier policy.
type Server struct {
	mcp       *mcp.Server
	client    *Client
	allow     tierSet
	staticKey string
}

// Options configures a Server.
type Options struct {
	Version string
	// AllowTiers overrides the default tier policy (reads + idempotent writes).
	// Nil means defaultTiers().
	AllowTiers tierSet
	// StaticKey, when set, is used as the API key for every call regardless of
	// request headers. This is the local single-tenant mode (stdio transport /
	// Claude Desktop), where the key comes from RECURSO_API_KEY rather than a
	// per-request Authorization header. Leave empty for the multi-tenant HTTP
	// transport, where each caller supplies their own key.
	StaticKey string
}

// NewServer builds the Recurso MCP server with the enabled tools registered.
// The MCP server is shared across HTTP requests; each tool call reads the
// caller's key off the request headers, so one server safely serves many
// tenants without ever holding a credential itself.
func NewServer(client *Client, opts Options) *Server {
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}
	allow := opts.AllowTiers
	if allow == nil {
		allow = defaultTiers()
	}
	s := &Server{
		mcp:       mcp.NewServer(&mcp.Implementation{Name: serverName, Version: opts.Version}, nil),
		client:    client,
		allow:     allow,
		staticKey: strings.TrimSpace(opts.StaticKey),
	}
	registerReadTools(s)
	registerWriteTools(s)
	registerSensitiveTools(s)
	return s
}

// MCP returns the underlying MCP server for transport wiring.
func (s *Server) MCP() *mcp.Server { return s.mcp }

// --- shared helpers used by tool handlers ---

// resolveKey determines the API key for a call. In local stdio mode a static
// key (from RECURSO_API_KEY) is used for every call. Otherwise the bearer key
// is taken from the request's Authorization header. It fails closed: a
// missing/blank key yields an error the handler surfaces as a tool error, so
// nothing is ever sent to /v1 unauthenticated.
func (s *Server) resolveKey(req *mcp.CallToolRequest) (string, *APIError) {
	if s.staticKey != "" {
		return s.staticKey, nil
	}
	if req == nil || req.Extra == nil || req.Extra.Header == nil {
		return "", &APIError{Status: http.StatusUnauthorized, Message: "missing credentials"}
	}
	h := req.Extra.Header.Get("Authorization")
	key := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	if key == "" {
		return "", &APIError{Status: http.StatusUnauthorized, Message: "missing API key — send an Authorization: Bearer rsk_... header"}
	}
	return key, nil
}

// jsonResult wraps a raw /v1 JSON body as a successful tool result.
func jsonResult(body []byte) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}
}

// errResult reports a tool-level failure to the agent (IsError) rather than a
// protocol error, so the model can read and react to it.
func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// addReadTool registers a Tier-1 GET tool if Tier1 is enabled. build maps the
// typed input to a (path, query); the handler forwards the caller's key.
func addReadTool[In any](s *Server, name, title, desc string, build func(In) (path string, query url.Values, err error)) {
	if !Tier1.enabled(s.allow) {
		return
	}
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        name,
		Description: desc,
		Annotations: Tier1.annotations(title),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		key, aerr := s.resolveKey(req)
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		path, query, err := build(in)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		body, aerr := s.client.Get(ctx, key, path, query)
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		return jsonResult(body), nil, nil
	})
}

// addSimTool registers a Tier-1 POST tool for read-only computations (e.g. the
// pricing simulator): it POSTs but persists nothing, so it carries the
// read-only annotation and sends no idempotency key.
func addSimTool[In any](s *Server, name, title, desc string, build func(In) (path string, body any, err error)) {
	if !Tier1.enabled(s.allow) {
		return
	}
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        name,
		Description: desc,
		Annotations: Tier1.annotations(title),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		key, aerr := s.resolveKey(req)
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		path, b, err := build(in)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		body, aerr := s.client.Post(ctx, key, path, b, "")
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		return jsonResult(body), nil, nil
	})
}

// addWriteTool registers a Tier-2 idempotent write tool if Tier2 is enabled.
// method is POST or PUT. build maps the typed input to (path, body, idemKey);
// when idemKey is empty a fresh one is generated so the request always carries
// an Idempotency-Key — an agent that wants retry-safety should pass a stable
// idempotency_key of its own.
func addWriteTool[In any](s *Server, method, name, title, desc string, build func(In) (path string, body any, idemKey string, err error)) {
	if !Tier2.enabled(s.allow) {
		return
	}
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        name,
		Description: desc,
		Annotations: Tier2.annotations(title),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		key, aerr := s.resolveKey(req)
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		path, b, idem, err := build(in)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		if idem == "" {
			idem = uuid.NewString()
		}
		var body []byte
		if method == http.MethodPut {
			body, aerr = s.client.Put(ctx, key, path, b, idem)
		} else {
			body, aerr = s.client.Post(ctx, key, path, b, idem)
		}
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		return jsonResult(body), nil, nil
	})
}
