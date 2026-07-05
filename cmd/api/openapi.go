package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// openAPISpecYAML is the OpenAPI 3.1 document for this API, embedded at build
// time. cmd/api/openapi.yaml is the single source of truth for the spec.
//
//go:embed openapi.yaml
var openAPISpecYAML []byte

// convertOpenAPISpecToJSON parses the embedded YAML spec and re-encodes it as
// JSON. gopkg.in/yaml.v3 decodes mappings into map[string]interface{}, so the
// result marshals to JSON directly.
func convertOpenAPISpecToJSON(specYAML []byte) ([]byte, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(specYAML, &doc); err != nil {
		return nil, fmt.Errorf("invalid YAML in embedded OpenAPI spec: %w", err)
	}
	if _, ok := doc["openapi"]; !ok {
		return nil, fmt.Errorf("embedded OpenAPI spec is missing the 'openapi' field")
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to encode OpenAPI spec as JSON: %w", err)
	}
	return out, nil
}

// registerOpenAPIRoutes serves the API specification publicly:
//
//	GET /openapi.yaml  — the spec as authored (application/yaml)
//	GET /openapi.json  — the same spec converted to JSON at startup
func registerOpenAPIRoutes(r *gin.Engine) error {
	specJSON, err := convertOpenAPISpecToJSON(openAPISpecYAML)
	if err != nil {
		return err
	}

	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", openAPISpecYAML)
	})
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", specJSON)
	})
	return nil
}
