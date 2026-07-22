// Package mcp implements the Recurso MCP (Model Context Protocol) server: a
// curated, tier-gated, agent-ergonomic facade over the existing /v1 HTTP API.
// It holds no database connection and no service structs — tenant isolation is
// the caller's API key, forwarded on every request (see docs/spec_mcp_server.md).
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodyBytes caps how much of a /v1 response we read, so a pathological
// upstream can't exhaust memory.
const maxBodyBytes = 4 << 20 // 4 MiB

// Client is a thin, tenant-agnostic HTTP client over the Recurso /v1 API. It
// holds NO credentials of its own: every call forwards the MCP caller's API
// key, so tenant scoping and live/test mode are decided entirely by that key at
// the /v1 layer.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Client targeting the given /v1 base URL (e.g.
// "https://api.recurso.dev").
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a /v1 failure surfaced to the agent as a tool error. It carries
// the upstream message only — never a raw stack or internal detail.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("request failed (HTTP %d)", e.Status)
	}
	return e.Message
}

// Get issues GET baseURL+path?query authenticated with the caller's key.
func (c *Client) Get(ctx context.Context, key, path string, query url.Values) ([]byte, *APIError) {
	return c.do(ctx, http.MethodGet, key, path, query, nil, "")
}

// Post issues POST baseURL+path with a JSON body and the caller's key. When
// idemKey is non-empty it is sent as Idempotency-Key so retries are safe.
func (c *Client) Post(ctx context.Context, key, path string, body any, idemKey string) ([]byte, *APIError) {
	return c.do(ctx, http.MethodPost, key, path, nil, body, idemKey)
}

func (c *Client) do(ctx context.Context, method, key, path string, query url.Values, body any, idemKey string) ([]byte, *APIError) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, &APIError{Message: "could not encode request"}
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, &APIError{Message: "could not build request"}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Message: "billing API unreachable"}
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Status: resp.StatusCode, Message: extractError(data, resp.StatusCode)}
	}
	return data, nil
}

// extractError pulls {"error":{"message":...}} out of a /v1 error body, falling
// back to a generic message keyed on the status.
func extractError(body []byte, status int) string {
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
		return env.Error.Message
	}
	switch status {
	case http.StatusUnauthorized:
		return "unauthorized — check the API key"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not found"
	default:
		return fmt.Sprintf("request failed (HTTP %d)", status)
	}
}
