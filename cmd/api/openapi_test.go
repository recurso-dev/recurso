package main

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestEmbeddedOpenAPISpecParses verifies the embedded spec is valid YAML,
// declares OpenAPI 3.1, and has a non-empty set of paths.
func TestEmbeddedOpenAPISpecParses(t *testing.T) {
	if len(openAPISpecYAML) == 0 {
		t.Fatal("embedded openapi.yaml is empty")
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(openAPISpecYAML, &doc); err != nil {
		t.Fatalf("embedded openapi.yaml is not valid YAML: %v", err)
	}

	version, ok := doc["openapi"].(string)
	if !ok || version == "" {
		t.Fatalf("missing or invalid 'openapi' field: %v", doc["openapi"])
	}
	if version[:3] != "3.1" {
		t.Errorf("expected OpenAPI 3.1.x, got %q", version)
	}

	paths, ok := doc["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		t.Fatal("'paths' must be a non-empty object")
	}

	// Spot-check a few core routes that must be documented.
	for _, p := range []string{
		"/auth/register",
		"/v1/plans",
		"/v1/customers",
		"/v1/subscriptions",
		"/v1/invoices",
		"/v1/quotes",
		"/v1/webhooks",
		"/v1/analytics/mrr",
		"/health",
	} {
		if _, ok := paths[p]; !ok {
			t.Errorf("expected path %q to be documented", p)
		}
	}
}

// TestOpenAPISpecJSONConversion verifies the YAML->JSON conversion used for
// GET /openapi.json produces valid JSON with the same top-level shape.
func TestOpenAPISpecJSONConversion(t *testing.T) {
	out, err := convertOpenAPISpecToJSON(openAPISpecYAML)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("converted spec is not valid JSON: %v", err)
	}
	if _, ok := doc["openapi"]; !ok {
		t.Error("converted JSON is missing 'openapi' field")
	}
	if paths, ok := doc["paths"].(map[string]interface{}); !ok || len(paths) == 0 {
		t.Error("converted JSON has empty 'paths'")
	}
}
