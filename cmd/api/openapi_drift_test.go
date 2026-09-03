package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestOpenAPISpecCoversRegisteredRoutes fails when a route registered in
// main.go / routes_v1.go has no corresponding entry in the embedded OpenAPI spec, so the
// documented surface can't silently drift from the served one (ENG-10).
//
// Routes are discovered by scanning main.go and routes_v1.go's gin registrations — the router
// is wired inside main(), so static scanning is the only way to enumerate it
// without refactoring startup. If a route is intentionally undocumented, add
// it to the allowlist below with a reason.
func TestOpenAPISpecCoversRegisteredRoutes(t *testing.T) {
	// Intentionally undocumented routes.
	allowlist := map[string]string{
		// none currently
	}

	// Route registrations live in main.go (public, auth, portal) and
	// routes_v1.go (the /v1 table); scan both so a route moved between them
	// is never lost from the gate.
	var src []byte
	for _, name := range []string{"main.go", "routes_v1.go"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src = append(src, b...)
		src = append(src, '\n')
	}

	// Group receivers → path prefixes (see main.go's r.Group calls).
	prefixes := map[string]string{
		"r":         "",
		"v1":        "/v1",
		"portal":    "/portal/api",
		"analytics": "/v1/analytics",
	}

	re := regexp.MustCompile(`(?m)^\s*(r|v1|portal|analytics)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(src), -1)
	if len(matches) < 100 {
		t.Fatalf("only %d route registrations found — the scanner regex is probably broken", len(matches))
	}

	// Parse the embedded spec's paths → methods.
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(openAPISpecYAML, &spec); err != nil {
		t.Fatalf("parse embedded spec: %v", err)
	}

	ginParam := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	var missing []string
	for _, m := range matches {
		group, method, ginPath := m[1], strings.ToLower(m[2]), m[3]
		path := prefixes[group] + ginParam.ReplaceAllString(ginPath, `{$1}`)
		key := method + " " + path
		if _, ok := allowlist[key]; ok {
			continue
		}
		ops, ok := spec.Paths[path]
		if !ok {
			missing = append(missing, fmt.Sprintf("%-6s %s (path absent)", strings.ToUpper(method), path))
			continue
		}
		if _, ok := ops[method]; !ok {
			missing = append(missing, fmt.Sprintf("%-6s %s (method absent)", strings.ToUpper(method), path))
		}
	}

	if len(missing) > 0 {
		t.Errorf("%d registered routes are missing from openapi.yaml:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}
