# recurso-go

Official Go client for the [Recurso](https://github.com/recurso-dev/recurso) billing API — a
hand-crafted, typed, resource-oriented client covering plans, customers, the subscription
lifecycle, invoices, the usage platform, quotes, entitlements, webhooks (with delivery tracking
and redelivery), analytics, and more.

- **Standard library only** — no third-party dependencies.
- Go 1.22+.
- Every method takes a `context.Context` first and returns `(T, error)`.
- Non-2xx responses come back as `*recurso.APIError` carrying the API's `{"error": {"code", "message"}}` envelope and the HTTP status.
- Monetary amounts are `int64` in the currency's smallest unit.

```bash
go get github.com/recurso-dev/recurso-go
```

## Usage

```go
package main

import (
	"context"
	"errors"
	"log"

	recurso "github.com/recurso-dev/recurso-go"
)

func main() {
	ctx := context.Background()
	client := recurso.NewClient(
		"rsk_test_your_api_key",
		recurso.WithBaseURL("https://billing.example.com/v1"), // default: https://api.recurso.dev/v1
	)

	plan, err := client.Plans.Create(ctx, &recurso.PlanCreateParams{
		Name:         "Pro Plan",
		Code:         "PRO-USD",
		Amount:       2900, // minor units ($29.00)
		Currency:     "USD",
		IntervalUnit: "month",
	})
	if err != nil {
		log.Fatal(err)
	}

	customer, err := client.Customers.Create(ctx, &recurso.CustomerCreateParams{
		Name:  "Jane User",
		Email: "jane@example.com",
		// Country defaults to "US" when omitted.
	})
	if err != nil {
		log.Fatal(err)
	}

	sub, err := client.Subscriptions.Create(ctx, &recurso.SubscriptionCreateParams{
		CustomerID: customer.ID,
		PlanID:     plan.ID,
	})
	if err != nil {
		// Any non-2xx response decodes to *recurso.APIError.
		var apiErr *recurso.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("recurso: %s (%s, HTTP %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
		}
		log.Fatal(err)
	}

	log.Printf("created subscription %s (%s)", sub.ID, sub.Status)
}
```

## Options

- `recurso.WithBaseURL(url)` — point at a self-hosted instance (include the `/v1` prefix).
- `recurso.WithHTTPClient(hc)` — supply a custom `*http.Client` (timeouts, transport, proxies).

## Notes

- Endpoints with a small or flexible body accept `recurso.Params` (`map[string]any`); the common
  create paths (plans, customers, subscriptions, coupons, usage events, webhooks) have typed
  `*…Params` structs.
- List endpoints accept `*recurso.ListParams` (`Page`, `Limit`, `Q`, `Status`) and return a typed
  slice.
- The client is safe for concurrent use.

Full method reference and guides: [docs.recurso.dev](https://docs.recurso.dev/sdk).

## Testing

```bash
cd sdk/go && go test ./...
```

## License

MIT
