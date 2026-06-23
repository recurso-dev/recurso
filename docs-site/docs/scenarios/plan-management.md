---
sidebar_position: 2
---

# Plan & Pricing Management

Plans define what you sell, how often you bill, and at what price. This guide covers creating plans, managing multi-currency pricing, and understanding billing intervals.

## The Plan Object

```json
{
  "id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "name": "Pro Plan",
  "code": "pro-monthly",
  "interval_unit": "month",
  "interval_count": 1,
  "prices": [
    {
      "id": "price_aabb1122-3344-5566-7788-99aabbccddee",
      "amount": 4900,
      "currency": "USD"
    },
    {
      "id": "price_ccdd1122-3344-5566-7788-99aabbccddee",
      "amount": 4500,
      "currency": "EUR"
    }
  ],
  "created_at": "2026-05-15T09:00:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique plan identifier. |
| `name` | string | Human-readable display name. |
| `code` | string | Unique slug for referencing in code or APIs. |
| `interval_unit` | string | The billing frequency unit: `day`, `week`, `month`, or `year`. |
| `interval_count` | integer | How many interval units make up one billing cycle. |
| `prices` | array | One or more Price objects, each with a different currency. |
| `created_at` | string | ISO 8601 timestamp of when the plan was created. |

---

## Create a Plan

Define a new plan with its billing interval and pricing.

**POST** `/v1/plans`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name of the plan. |
| `code` | string | No | Unique slug (e.g., `pro-monthly`). Auto-generated if omitted. |
| `interval_unit` | string | Yes | One of `day`, `week`, `month`, or `year`. |
| `interval_count` | integer | Yes | Number of units per billing cycle (e.g., `1` for monthly, `3` for quarterly). |
| `amount` | integer | Yes | Price in the smallest currency unit (cents). For example, `4900` = $49.00. |
| `currency` | string | Yes | Three-letter ISO 4217 currency code (e.g., `USD`, `EUR`, `INR`). |

### Example: Monthly Plan

```bash
curl -X POST https://api.recurso.dev/v1/plans \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan",
    "code": "pro-monthly",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 4900,
    "currency": "USD"
  }'
```

Response:
```json
{
  "id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
  "name": "Pro Plan",
  "code": "pro-monthly",
  "interval_unit": "month",
  "interval_count": 1,
  "prices": [
    {
      "id": "price_aabb1122-3344-5566-7788-99aabbccddee",
      "amount": 4900,
      "currency": "USD"
    }
  ],
  "created_at": "2026-06-23T11:05:00Z"
}
```

### Example: Yearly Plan

```bash
curl -X POST https://api.recurso.dev/v1/plans \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Enterprise Annual",
    "code": "enterprise-annual",
    "interval_unit": "year",
    "interval_count": 1,
    "amount": 199900,
    "currency": "USD"
  }'
```

Response:
```json
{
  "id": "plan_ee44ff55-6677-8899-aabb-ccddeeff0011",
  "name": "Enterprise Annual",
  "code": "enterprise-annual",
  "interval_unit": "year",
  "interval_count": 1,
  "prices": [
    {
      "id": "price_1122aabb-ccdd-eeff-0011-223344556677",
      "amount": 199900,
      "currency": "USD"
    }
  ],
  "created_at": "2026-06-23T11:10:00Z"
}
```

---

## List Plans

Retrieve all plans for your tenant.

**GET** `/v1/plans`

```bash
curl https://api.recurso.dev/v1/plans \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
      "name": "Pro Plan",
      "code": "pro-monthly",
      "interval_unit": "month",
      "interval_count": 1,
      "prices": [
        {
          "id": "price_aabb1122-3344-5566-7788-99aabbccddee",
          "amount": 4900,
          "currency": "USD"
        }
      ],
      "created_at": "2026-06-23T11:05:00Z"
    },
    {
      "id": "plan_ee44ff55-6677-8899-aabb-ccddeeff0011",
      "name": "Enterprise Annual",
      "code": "enterprise-annual",
      "interval_unit": "year",
      "interval_count": 1,
      "prices": [
        {
          "id": "price_1122aabb-ccdd-eeff-0011-223344556677",
          "amount": 199900,
          "currency": "USD"
        }
      ],
      "created_at": "2026-06-23T11:10:00Z"
    }
  ],
  "has_more": false
}
```

---

## Multi-Currency Pricing

A single plan can have multiple **Price** objects, each in a different currency. This allows you to offer localized pricing to customers around the world without creating separate plans.

When a subscription is created, Recurso selects the Price that matches the customer's currency. If no match is found, the default currency price is used.

### The Price Object

```json
{
  "id": "price_aabb1122-3344-5566-7788-99aabbccddee",
  "amount": 4900,
  "currency": "USD"
}
```

### Example: Adding a Plan with Multiple Currencies

Create the plan with its primary currency first, then add additional prices:

```bash
curl -X POST https://api.recurso.dev/v1/plans \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Startup Plan",
    "code": "startup-monthly",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 2900,
    "currency": "USD"
  }'
```

You can include multiple price objects when your plan supports multi-currency. The plan response will contain all configured prices in the `prices` array:

```json
{
  "id": "plan_ff001122-3344-5566-7788-99aabbccddee",
  "name": "Startup Plan",
  "code": "startup-monthly",
  "interval_unit": "month",
  "interval_count": 1,
  "prices": [
    { "id": "price_a1001122-3344-5566-7788-99aabbccddee", "amount": 2900, "currency": "USD" },
    { "id": "price_b2001122-3344-5566-7788-99aabbccddee", "amount": 2700, "currency": "EUR" },
    { "id": "price_c3001122-3344-5566-7788-99aabbccddee", "amount": 239900, "currency": "INR" }
  ],
  "created_at": "2026-06-23T11:30:00Z"
}
```

---

## Interval Types Explained

The combination of `interval_unit` and `interval_count` gives you full flexibility over billing cycles:

| interval_unit | interval_count | Billing Cycle |
|---|---|---|
| `day` | `1` | Daily |
| `day` | `7` | Every 7 days |
| `week` | `1` | Weekly |
| `week` | `2` | Biweekly |
| `month` | `1` | Monthly |
| `month` | `3` | Quarterly |
| `month` | `6` | Semi-annually |
| `year` | `1` | Annually |
| `year` | `2` | Biennially |

Choose the combination that matches your pricing model. Most SaaS businesses use `month/1` (monthly) and `year/1` (annual) plans.
