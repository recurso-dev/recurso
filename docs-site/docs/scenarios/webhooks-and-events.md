---
sidebar_position: 13
---

# Webhooks & Events

Recurso fires events whenever something significant happens in your account -- a subscription is created, an invoice is paid, a payment fails, and more. You can register webhook endpoints to receive these events in real time, enabling you to build automated workflows, sync data with external systems, and react to billing changes instantly.

---

## How Webhooks Work

1. **Register a webhook endpoint** with the URL where you want to receive events and the event types you care about.
2. **Recurso fires an event** when the corresponding action occurs (e.g., a subscription is created).
3. **Recurso delivers the event** to all registered endpoints that subscribe to that event type via an HTTP POST request.
4. **Your server acknowledges** by responding with a `2xx` status code.
5. **Failed deliveries are retried** with exponential backoff (1 min, 5 min, 30 min, 2 hours, 24 hours) for up to 72 hours.

Each delivery includes an HMAC-SHA256 signature in the `X-Recurso-Signature` header, computed using the endpoint's secret. Always verify this signature before processing the payload.

---

## The Webhook Endpoint Object

```json
{
  "id": "wh_4e5f6a7b-8c9d-0e1f-2a3b-4c5d6e7f8a9b",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "url": "https://example.com/webhooks/recurso",
  "events": [
    "subscription.created",
    "invoice.paid",
    "payment.failed"
  ],
  "secret": "whsec_a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "active": true,
  "created_at": "2026-05-01T10:00:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique webhook endpoint identifier. |
| `url` | string | The URL where events are delivered via HTTP POST. |
| `events` | array | List of event types this endpoint subscribes to. |
| `secret` | string | The HMAC secret used to sign deliveries. Store this securely. |
| `active` | boolean | Whether the endpoint is currently receiving events. |
| `created_at` | string | ISO 8601 timestamp of when the endpoint was created. |

---

## Create a Webhook Endpoint

Register a new URL to receive events.

**POST** `/v1/webhooks`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `url` | string | Yes | The HTTPS URL that will receive event payloads. |
| `events` | array | Yes | List of event types to subscribe to (e.g., `["subscription.created", "invoice.paid"]`). |
| `secret` | string | No | A custom HMAC secret. If omitted, Recurso generates one automatically. |

```bash
curl -X POST https://api.recurso.dev/v1/webhooks \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/webhooks/recurso",
    "events": ["subscription.created", "invoice.paid", "payment.failed"],
    "secret": "whsec_a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
  }'
```

Response:
```json
{
  "id": "wh_4e5f6a7b-8c9d-0e1f-2a3b-4c5d6e7f8a9b",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "url": "https://example.com/webhooks/recurso",
  "events": [
    "subscription.created",
    "invoice.paid",
    "payment.failed"
  ],
  "secret": "whsec_a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "active": true,
  "created_at": "2026-06-23T14:00:00Z"
}
```

---

## List Webhook Endpoints

Retrieve all registered webhook endpoints for your tenant.

**GET** `/v1/webhooks`

```bash
curl https://api.recurso.dev/v1/webhooks \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "wh_4e5f6a7b-8c9d-0e1f-2a3b-4c5d6e7f8a9b",
      "url": "https://example.com/webhooks/recurso",
      "events": ["subscription.created", "invoice.paid", "payment.failed"],
      "active": true,
      "created_at": "2026-06-23T14:00:00Z"
    },
    {
      "id": "wh_9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
      "url": "https://example.com/webhooks/slack-alerts",
      "events": ["payment.failed", "subscription.cancelled"],
      "active": true,
      "created_at": "2026-05-15T09:30:00Z"
    }
  ],
  "has_more": false
}
```

---

## Delete a Webhook Endpoint

Remove a webhook endpoint. Events will no longer be delivered to this URL.

**DELETE** `/v1/webhooks/:id`

```bash
curl -X DELETE https://api.recurso.dev/v1/webhooks/wh_9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "wh_9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
  "deleted": true
}
```

---

## Events

Every action in Recurso that changes state produces an event. Events are immutable records that can be listed, inspected, and replayed.

### The Event Object

```json
{
  "id": "evt_c1d2e3f4-a5b6-7890-cdef-1234567890ab",
  "type": "invoice.paid",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "data": {
    "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "status": "paid",
    "total": 5782,
    "currency": "USD",
    "paid_at": "2026-06-23T14:30:00Z"
  },
  "created_at": "2026-06-23T14:30:01Z"
}
```

### List Events

Retrieve a paginated list of events for your tenant. Use this to audit activity or debug webhook deliveries.

**GET** `/v1/events`

```bash
curl https://api.recurso.dev/v1/events \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "evt_c1d2e3f4-a5b6-7890-cdef-1234567890ab",
      "type": "invoice.paid",
      "data": {
        "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
        "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
        "status": "paid",
        "total": 5782,
        "currency": "USD",
        "paid_at": "2026-06-23T14:30:00Z"
      },
      "created_at": "2026-06-23T14:30:01Z"
    },
    {
      "id": "evt_d2e3f4a5-b6c7-8901-defa-234567890bcd",
      "type": "subscription.created",
      "data": {
        "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
        "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
        "status": "active",
        "current_period_start": "2026-06-23T00:00:00Z",
        "current_period_end": "2026-07-23T00:00:00Z"
      },
      "created_at": "2026-06-23T14:00:05Z"
    },
    {
      "id": "evt_e3f4a5b6-c7d8-9012-efab-345678901cde",
      "type": "payment.failed",
      "data": {
        "id": "pay_f4a5b6c7-d8e9-0123-fabc-456789012def",
        "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
        "invoice_id": "inv_a5b6c7d8-e9f0-1234-abcd-567890123456",
        "amount": 199900,
        "currency": "INR",
        "failure_reason": "insufficient_funds"
      },
      "created_at": "2026-06-22T08:15:00Z"
    }
  ],
  "has_more": true
}
```

### Get Event Types

Retrieve the full list of event types that Recurso can fire. Use this to discover which events are available when configuring your webhook endpoints.

**GET** `/v1/events/types`

```bash
curl https://api.recurso.dev/v1/events/types \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    "subscription.created",
    "subscription.updated",
    "subscription.cancelled",
    "subscription.renewed",
    "subscription.paused",
    "subscription.resumed",
    "invoice.created",
    "invoice.paid",
    "invoice.voided",
    "invoice.uncollectible",
    "payment.succeeded",
    "payment.failed",
    "payment.refunded",
    "customer.created",
    "customer.updated",
    "credit_note.created",
    "coupon.applied",
    "quote.accepted",
    "quote.declined",
    "gift.purchased",
    "gift.redeemed"
  ]
}
```

---

## Supported Event Types

| Event Type | Description |
|---|---|
| `subscription.created` | A new subscription was created. |
| `subscription.updated` | A subscription was modified (plan change, quantity update, etc.). |
| `subscription.cancelled` | A subscription was cancelled. |
| `subscription.renewed` | A subscription was renewed for a new billing period. |
| `subscription.paused` | A subscription was paused. |
| `subscription.resumed` | A paused subscription was resumed. |
| `invoice.created` | A new invoice was generated. |
| `invoice.paid` | An invoice was paid in full. |
| `invoice.voided` | An invoice was voided. |
| `invoice.uncollectible` | An invoice was marked as uncollectible after failed retries. |
| `payment.succeeded` | A payment was successfully processed. |
| `payment.failed` | A payment attempt failed. |
| `payment.refunded` | A payment was refunded. |
| `customer.created` | A new customer was created. |
| `customer.updated` | A customer's details were updated. |
| `credit_note.created` | A credit note was issued. |
| `coupon.applied` | A coupon was applied to a subscription. |
| `quote.accepted` | A quote was accepted by the customer. |
| `quote.declined` | A quote was declined by the customer. |
| `gift.purchased` | A gift subscription was purchased. |
| `gift.redeemed` | A gift subscription was redeemed. |

---

## Verifying Webhook Signatures

Every webhook delivery includes an `X-Recurso-Signature` header containing an HMAC-SHA256 signature. You should always verify this signature to ensure the payload was sent by Recurso and has not been tampered with.

### How to Verify

1. Extract the `X-Recurso-Signature` header from the incoming request.
2. Compute an HMAC-SHA256 digest of the raw request body using your endpoint's `secret`.
3. Compare the computed digest with the signature from the header using a timing-safe comparison.

### Example: Node.js Verification

```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}

// In your webhook handler:
app.post('/webhooks/recurso', (req, res) => {
  const signature = req.headers['x-recurso-signature'];
  const isValid = verifySignature(req.rawBody, signature, 'whsec_a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6');

  if (!isValid) {
    return res.status(401).json({ error: 'Invalid signature' });
  }

  const event = JSON.parse(req.rawBody);
  console.log(`Received event: ${event.type}`);

  // Process the event...
  res.status(200).json({ received: true });
});
```

---

## Retry Policy

When a webhook delivery fails (non-2xx response or timeout), Recurso retries with exponential backoff:

| Attempt | Delay After Failure |
|---|---|
| 1st retry | 1 minute |
| 2nd retry | 5 minutes |
| 3rd retry | 30 minutes |
| 4th retry | 2 hours |
| 5th retry | 24 hours |

After 5 failed retries (approximately 72 hours total), the delivery is marked as failed. You can inspect failed deliveries in the Events list and replay them manually.

---

## Best Practices

- **Always verify signatures.** Never process a webhook payload without verifying the `X-Recurso-Signature` header.
- **Respond quickly.** Return a `200` status code as soon as possible. Process the event asynchronously (e.g., add it to a queue) to avoid timeouts.
- **Handle duplicates.** Recurso may deliver the same event more than once during retries. Use the event `id` to deduplicate.
- **Subscribe only to the events you need.** This reduces noise and processing overhead.
- **Monitor your endpoints.** If deliveries are consistently failing, check your server logs and ensure the URL is reachable.
