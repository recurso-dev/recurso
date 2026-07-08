# recurso-go

Official Go SDK for the [Recurso](https://github.com/swapnull-in/recur-so) billing API — 18 resources / 68 methods covering plans, customers, the full subscription lifecycle (pause/resume/cancel, add-ons, plan-change preview), invoices, usage, coupons, quotes, entitlements, webhooks (deliveries and redelivery), events, credit notes, gifts, referrals, mandates, ledger, analytics, developer keys, and the tenant account.

Standard library only — no third-party dependencies. Requires Go 1.22+.

The client is hand-crafted but generated-in-spirit from the OpenAPI 3.1 description in [`cmd/api/openapi.yaml`](../../cmd/api/openapi.yaml): types and paths mirror the spec exactly, and monetary amounts are `int64` values in the currency's smallest unit (cents/paise).

## Install

```bash
go get github.com/swapnull-in/recurso-go
```

## Quickstart

```go
package main

import (
	"context"
	"errors"
	"log"

	recurso "github.com/swapnull-in/recurso-go"
)

func main() {
	ctx := context.Background()
	client := recurso.NewClient("rsk_live_your_api_key")

	plan, err := client.Plans.Create(ctx, &recurso.PlanCreateParams{
		Name:          "Pro Plan",
		Code:          "PRO-USD",
		Amount:        2900, // minor units ($29.00)
		Currency:      "USD",
		IntervalUnit:  "month",
		IntervalCount: 1,
	})
	if err != nil {
		log.Fatal(err)
	}

	customer, err := client.Customers.Create(ctx, &recurso.CustomerCreateParams{
		Name:    "Jane User",
		Email:   "jane@example.com",
		Country: "US",
	})
	if err != nil {
		log.Fatal(err)
	}

	sub, err := client.Subscriptions.Create(ctx, &recurso.SubscriptionCreateParams{
		CustomerID: customer.ID,
		PlanID:     plan.ID,
	})
	if err != nil {
		// See "Error handling" below.
		var apiErr *recurso.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("recurso: %s (%s, HTTP %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
		}
		log.Fatal(err)
	}

	log.Printf("created subscription %s (%s)", sub.ID, sub.Status)
}
```

## Authentication

Every request carries your API key as a bearer token
(`Authorization: Bearer <key>`). Pass the key to `NewClient`:

```go
client := recurso.NewClient("rsk_live_your_api_key")
```

## Configuration

`NewClient` accepts functional options:

```go
client := recurso.NewClient(
	"rsk_live_your_api_key",
	recurso.WithBaseURL("https://billing.example.com/v1"), // default: https://api.recurso.dev/v1
	recurso.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
)
```

- `WithBaseURL` targets a self-hosted or staging deployment. The base URL
  includes the `/v1` version prefix.
- `WithHTTPClient` supplies a custom `*http.Client` for timeouts, proxies, or
  instrumentation.

## Resource layout

Every endpoint hangs off a resource field on the `Client`, e.g.
`client.Plans`, `client.Customers`, `client.Subscriptions`, `client.Invoices`,
`client.Usage`, `client.Coupons`, `client.Quotes`, `client.Entitlements`,
`client.Webhooks`, `client.Events`, `client.CreditNotes`, `client.Gifts`,
`client.Referrals`, `client.Mandates`, `client.Ledger`, `client.Analytics`,
`client.Developer`, and `client.Account`. Every method takes a
`context.Context` first and returns `(T, error)`.

## Error handling

Any non-2xx response is returned as a `*recurso.APIError`, which decodes the
standard error envelope (`{"error": {"code", "message"}}`) and carries the HTTP
status code:

```go
_, err := client.Plans.List(ctx, nil)
var apiErr *recurso.APIError
if errors.As(err, &apiErr) {
	log.Printf("code=%s status=%d: %s", apiErr.Code, apiErr.StatusCode, apiErr.Message)
}
```

## Testing

The suite in `recurso_test.go` uses `net/http/httptest` to mock the API,
asserting request method/path/auth/body and typed response parsing across every
resource, plus the error path. Run it with:

```bash
go test ./...
```

## License

MIT
