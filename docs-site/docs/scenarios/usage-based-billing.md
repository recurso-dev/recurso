---
sidebar_position: 5
---

# Usage-Based Billing

Recurso supports usage-based (metered) billing alongside fixed recurring charges. This guide covers recording usage events, adding unbilled charges, and generating advance invoices.

## How the Metering Model Works

Usage-based billing in Recurso follows this flow:

1. **Record events** -- Your application sends usage events to Recurso as they occur (e.g., API calls, storage consumed, messages sent).
2. **Aggregate** -- Recurso aggregates events over the billing period using the metric you define (sum, count, max).
3. **Invoice** -- At the end of the billing period, Recurso calculates the total usage, applies your per-unit pricing, and adds the charges to the subscription invoice.

You can also manually add one-off charges to a subscription that will appear on the next invoice.

---

## Record Usage Events

Send usage events to Recurso as they happen. Events are idempotent when you provide a unique identifier.

**POST** `/v1/usage/events`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `subscription_id` | string | Yes | The subscription to attribute usage to. |
| `metric` | string | Yes | The name of the usage metric (e.g., `api_calls`, `storage_gb`, `messages_sent`). |
| `quantity` | number | Yes | The quantity of usage to record. |
| `timestamp` | string | No | ISO 8601 timestamp of when the usage occurred. Defaults to now. |

### Example: Record API Call Usage

```bash
curl -X POST https://api.recurso.dev/v1/usage/events \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "metric": "api_calls",
    "quantity": 150,
    "timestamp": "2026-06-23T14:30:00Z"
  }'
```

Response:
```json
{
  "id": "evt_7788aabb-ccdd-eeff-0011-223344556677",
  "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "metric": "api_calls",
  "quantity": 150,
  "timestamp": "2026-06-23T14:30:00Z",
  "created_at": "2026-06-23T14:30:01Z"
}
```

### Example: Record Storage Usage

```bash
curl -X POST https://api.recurso.dev/v1/usage/events \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "metric": "storage_gb",
    "quantity": 2.5,
    "timestamp": "2026-06-23T15:00:00Z"
  }'
```

Response:
```json
{
  "id": "evt_99001122-3344-5566-7788-aabbccddeeff",
  "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "metric": "storage_gb",
  "quantity": 2.5,
  "timestamp": "2026-06-23T15:00:00Z",
  "created_at": "2026-06-23T15:00:01Z"
}
```

### Tips for Event Ingestion

- **Send events in real time** or in small batches. Recurso processes events asynchronously.
- **Use meaningful metric names.** Metric names are case-sensitive. Use consistent naming like `api_calls`, not `API_Calls` or `apiCalls`.
- **Include timestamps** when backfilling historical data. If omitted, the current server time is used.
- **Events are immutable.** Once recorded, events cannot be modified or deleted.

---

## Add Unbilled Charges

Add one-off charges to a subscription. These charges appear on the next invoice alongside any recurring and usage charges.

**POST** `/v1/subscriptions/:id/charges`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `description` | string | Yes | A description of the charge. |
| `amount` | integer | Yes | Amount in cents. |
| `currency` | string | No | Currency code. Defaults to the subscription's currency. |

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/charges \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Premium support add-on (June 2026)",
    "amount": 5000
  }'
```

Response:
```json
{
  "id": "chrg_dd112233-4455-6677-8899-aabbccddeeff",
  "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "description": "Premium support add-on (June 2026)",
  "amount": 5000,
  "currency": "USD",
  "invoiced": false,
  "created_at": "2026-06-23T16:00:00Z"
}
```

---

## List Unbilled Charges

View all pending charges for a subscription that have not yet been invoiced.

**GET** `/v1/subscriptions/:id/charges`

```bash
curl https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/charges \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "chrg_dd112233-4455-6677-8899-aabbccddeeff",
      "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "description": "Premium support add-on (June 2026)",
      "amount": 5000,
      "currency": "USD",
      "invoiced": false,
      "created_at": "2026-06-23T16:00:00Z"
    },
    {
      "id": "chrg_ff001122-3344-5566-7788-99aabb001122",
      "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "description": "Data migration service",
      "amount": 15000,
      "currency": "USD",
      "invoiced": false,
      "created_at": "2026-06-20T10:00:00Z"
    }
  ],
  "has_more": false
}
```

---

## Generate an Advance Invoice

Normally, usage charges are invoiced at the end of the billing period. Use the advance invoice endpoint to generate an invoice immediately, including all unbilled usage and charges accumulated so far.

This is useful when:
- A customer wants to pay for usage mid-cycle.
- You need to collect payment before the period ends.
- A subscription is being canceled and you want to bill for accrued usage.

**POST** `/v1/subscriptions/:id/advance`

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/advance \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "inv_aabb3344-5566-7788-99ee-ff0011223344",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "open",
  "subtotal": 20000,
  "tax_amount": 3600,
  "total": 23600,
  "currency": "USD",
  "line_items": [
    {
      "description": "Premium support add-on (June 2026)",
      "quantity": 1,
      "unit_amount": 5000,
      "amount": 5000
    },
    {
      "description": "Data migration service",
      "quantity": 1,
      "unit_amount": 15000,
      "amount": 15000
    }
  ],
  "created_at": "2026-06-23T17:00:00Z"
}
```

---

## Example: Full Usage-Based Billing Flow

Here is a complete workflow for billing a customer based on API call usage:

### 1. Create a Subscription

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321"
  }'
```

### 2. Record Usage Throughout the Period

```bash
curl -X POST https://api.recurso.dev/v1/usage/events \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "metric": "api_calls",
    "quantity": 500,
    "timestamp": "2026-06-15T12:00:00Z"
  }'
```

### 3. Recurso Generates the Invoice

At the end of the billing period, Recurso automatically totals all usage events for each metric, applies pricing, and generates an invoice that includes both the base plan charge and the metered usage charges.

### 4. (Optional) Generate an Advance Invoice

If you need to bill before the period ends:

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/advance \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```
