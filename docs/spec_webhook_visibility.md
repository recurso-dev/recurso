# Spec: Webhook Consumption & Visibility

## Objective
The platform needs to consume inbound webhook events from payment gateways (specifically refunds like `charge.refunded`) to auto-advance refund states internally. Additionally, the dashboard requires UI visibility into outbound webhooks, including delivery attempts, dead-letter status, and manual redelivery options.

## Tech Stack
- Go 1.25+
- React (Vite) for the frontend dashboard
- PostgreSQL (storing webhook events and delivery attempts)

## Commands
Build API: `make build`
Run UI: `cd frontend && npm run dev`
Test: `go test ./internal/adapter/handler/...`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  core/
    domain/             → `Refund` model state changes, `WebhookDelivery` model
  service/              → Update `WebhookService` to handle incoming refunds
  adapter/
    handler/            → `webhook.go` handling incoming Stripe/Razorpay refunds
frontend/
  src/
    pages/
      Webhooks.tsx      → UI for viewing outbound webhook delivery logs
    components/
      WebhookList.tsx   → List of webhooks
```

## Code Style
```go
// Handling incoming refund
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	// ... validation
	switch event.Type {
	case "charge.refunded":
		err := h.refundService.MarkRefundProcessed(ctx, event.Data.Object.ID)
		// ...
	}
}
```
```tsx
// Frontend representation
const WebhookDeliveryStatus = ({ status }) => {
  return <Badge color={status === 'success' ? 'green' : 'red'}>{status}</Badge>;
}
```

## Testing Strategy
- **Unit Tests**: Test the new case in the webhook handler (`charge.refunded`) to ensure it correctly identifies the internal refund record and advances its state.
- **Frontend Tests**: Ensure the webhook list component correctly renders success/failure states and redelivery buttons.
- **Integration Tests**: Simulate an incoming Stripe webhook payload and assert the database state changes.

## Boundaries
- **Always**: Verify the signature of incoming webhooks to prevent spoofing. Store the raw payload for auditability before processing.
- **Ask first**: Before adding a dedicated queuing system (like RabbitMQ or Redis Streams) for outbound webhooks; attempt to build the MVP using PostgreSQL-backed workers.
- **Never**: Block the HTTP response to the payment gateway while processing internal business logic; queue it or process it fast enough to meet the gateway's timeout requirements.

## Success Criteria
- [ ] Processing a Stripe/Razorpay refund via their dashboard results in the Recurso refund state automatically updating to "processed".
- [ ] Users can view a log of outbound webhooks sent to their configured endpoints in the dashboard.
- [ ] Users can click a "Redeliver" button on a failed outbound webhook, which schedules a new attempt.

## Open Questions
- Should webhook delivery logs be retained indefinitely, or automatically pruned after 30/90 days?
