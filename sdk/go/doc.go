// Package recurso is the official Go SDK for the Recurso billing API.
//
// It is a hand-crafted, idiomatic client generated in spirit from the
// OpenAPI 3.1 description in cmd/api/openapi.yaml. The surface mirrors the
// official Node SDK: methods are grouped per resource on a single Client,
// every call takes a context.Context first, and every response is a typed
// struct. Monetary amounts are int64 values in the currency's smallest unit
// (e.g. cents or paise), matching the API.
//
// # Quick start
//
//	client := recurso.NewClient("rsk_live_your_api_key")
//
//	customer, err := client.Customers.Create(ctx, &recurso.CustomerCreateParams{
//		Email:   "jane@example.com",
//		Name:    "Jane User",
//		Country: "US",
//	})
//	if err != nil {
//		var apiErr *recurso.APIError
//		if errors.As(err, &apiErr) {
//			log.Fatalf("recurso: %s (%s)", apiErr.Message, apiErr.Code)
//		}
//		log.Fatal(err)
//	}
//
// # Authentication
//
// Every request carries the API key as a bearer token
// (Authorization: Bearer <key>). Obtain a key from your Recurso dashboard or
// via the Developer resource.
//
// # Configuration
//
// NewClient accepts functional options: WithBaseURL to point at a different
// deployment (default https://api.recurso.dev/v1) and WithHTTPClient to
// supply a custom *http.Client (for timeouts, proxies, or instrumentation).
package recurso
