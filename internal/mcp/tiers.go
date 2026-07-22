package mcp

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Tier classifies a tool by its side-effect risk. The tier a tool sits in is a
// safety contract, not a suggestion (see docs/spec_mcp_server.md, decision D3):
//
//	Tier1 — reads & simulations, zero side effects, always exposed.
//	Tier2 — curated idempotent writes, on by default.
//	Tier3 — money-path / destructive, off until the tenant opts in.
type Tier int

const (
	Tier1 Tier = iota + 1
	Tier2
	Tier3
)

// tierSet is the set of tiers a server is allowed to expose.
type tierSet map[Tier]bool

// defaultTiers is the out-of-the-box policy: reads + idempotent writes on,
// money-path off. Inc 1 only registers Tier1; the gate exists here so Inc 2/3
// plug in without reworking registration.
func defaultTiers() tierSet {
	return tierSet{Tier1: true, Tier2: true, Tier3: false}
}

// enabled reports whether a tool of this tier may be registered under the given
// policy.
func (t Tier) enabled(allow tierSet) bool { return allow[t] }

func boolPtr(b bool) *bool { return &b }

// annotations builds MCP tool annotations that honestly reflect a tool's tier,
// so clients can surface read-only / idempotent / destructive hints to users.
func (t Tier) annotations(title string) *mcp.ToolAnnotations {
	switch t {
	case Tier1:
		return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true}
	case Tier2:
		return &mcp.ToolAnnotations{Title: title, IdempotentHint: true, DestructiveHint: boolPtr(false)}
	case Tier3:
		return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(true)}
	default:
		return &mcp.ToolAnnotations{Title: title}
	}
}
