package mcp

import (
	"os"
	"strings"
	"testing"
)

// TestReadToolPaths_ExistInOpenAPI guards against a tool pointing at a route
// that has been renamed or removed: every Tier-1 tool's /v1 path template must
// appear as a path key in the API's OpenAPI spec. This is the MCP-side analogue
// of the API's own openapi drift test.
func TestReadToolPaths_ExistInOpenAPI(t *testing.T) {
	data, err := os.ReadFile("../../cmd/api/openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	spec := string(data)
	check := func(name, path string) {
		// Path keys are indented two spaces under `paths:`.
		if !strings.Contains(spec, "\n  "+path+":") {
			t.Errorf("tool %q → %q not found as a path in cmd/api/openapi.yaml", name, path)
		}
	}
	for name, path := range readToolPaths {
		check(name, path)
	}
	for name, path := range writeToolPaths {
		check(name, path)
	}
	for name, path := range sensitiveToolPaths {
		check(name, path)
	}
	check("mcp_settings_optin", "/v1/settings/mcp") // the opt-in endpoint the Tier-3 gate calls
}
