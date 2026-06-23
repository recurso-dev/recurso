---
sidebar_position: 6
---

# Coupons & Promotions

Coupons let you offer discounts to customers. You can apply them as a percentage off or a fixed amount, and control how many times each coupon can be redeemed.

## The Coupon Object

```json
{
  "id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "code": "SUMMER20",
  "discount_type": "percent",
  "discount_value": 20,
  "max_redemptions": 100,
  "times_redeemed": 14,
  "active": true,
  "created_at": "2026-05-01T10:00:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique coupon identifier. |
| `code` | string | The code customers use to redeem the coupon. |
| `discount_type` | string | Either `percent` or `fixed`. |
| `discount_value` | number | The discount amount. For `percent`, this is a percentage (e.g., `20` = 20% off). For `fixed`, this is an amount in cents (e.g., `1000` = $10.00 off). |
| `max_redemptions` | integer | Maximum number of times this coupon can be used. `null` means unlimited. |
| `times_redeemed` | integer | How many times this coupon has been applied. |
| `active` | boolean | Whether the coupon is currently available for redemption. |
| `created_at` | string | ISO 8601 timestamp of when the coupon was created. |

---

## Create a Coupon

**POST** `/v1/coupons`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `code` | string | Yes | Unique coupon code (e.g., `SUMMER20`, `LAUNCH50OFF`). |
| `discount_type` | string | Yes | `percent` or `fixed`. |
| `discount_value` | number | Yes | Discount amount. Percentage (1-100) for `percent`, cents for `fixed`. |
| `max_redemptions` | integer | No | Maximum number of redemptions. Omit for unlimited. |

### Example: Percentage Discount

```bash
curl -X POST https://api.recurso.dev/v1/coupons \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "SUMMER20",
    "discount_type": "percent",
    "discount_value": 20,
    "max_redemptions": 100
  }'
```

Response:
```json
{
  "id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
  "code": "SUMMER20",
  "discount_type": "percent",
  "discount_value": 20,
  "max_redemptions": 100,
  "times_redeemed": 0,
  "active": true,
  "created_at": "2026-06-23T10:00:00Z"
}
```

### Example: Fixed Amount Discount

```bash
curl -X POST https://api.recurso.dev/v1/coupons \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "SAVE10",
    "discount_type": "fixed",
    "discount_value": 1000,
    "max_redemptions": 50
  }'
```

Response:
```json
{
  "id": "cpn_11223344-5566-7788-99aa-bbccddeeff00",
  "code": "SAVE10",
  "discount_type": "fixed",
  "discount_value": 1000,
  "max_redemptions": 50,
  "times_redeemed": 0,
  "active": true,
  "created_at": "2026-06-23T10:15:00Z"
}
```

---

## List Coupons

Retrieve all coupons for your tenant.

**GET** `/v1/coupons`

```bash
curl https://api.recurso.dev/v1/coupons \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
      "code": "SUMMER20",
      "discount_type": "percent",
      "discount_value": 20,
      "max_redemptions": 100,
      "times_redeemed": 14,
      "active": true,
      "created_at": "2026-06-23T10:00:00Z"
    },
    {
      "id": "cpn_11223344-5566-7788-99aa-bbccddeeff00",
      "code": "SAVE10",
      "discount_type": "fixed",
      "discount_value": 1000,
      "max_redemptions": 50,
      "times_redeemed": 3,
      "active": true,
      "created_at": "2026-06-23T10:15:00Z"
    }
  ],
  "has_more": false
}
```

---

## Applying a Coupon to a Subscription

Coupons are applied at subscription creation time using the `coupon_code` field. The discount is applied to every invoice generated for that subscription for its duration.

### Example: Create a Subscription with a Coupon

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
    "coupon_code": "SUMMER20"
  }'
```

Response:
```json
{
  "id": "sub_ee44ff55-0011-2233-4455-667788990011",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
  "status": "active",
  "current_period_start": "2026-06-23T00:00:00Z",
  "current_period_end": "2026-07-23T00:00:00Z",
  "coupon_id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
  "created_at": "2026-06-23T14:00:00Z"
}
```

### How the Discount Is Applied

When an invoice is generated for a subscription with a coupon:

- **Percent discount**: The discount is calculated as a percentage of the subtotal. For example, a 20% coupon on a $49.00 plan results in a $9.80 discount. The invoice subtotal becomes $39.20.
- **Fixed discount**: The fixed amount is subtracted from the subtotal. For example, a $10.00 coupon on a $49.00 plan results in a $39.00 subtotal.

The discount is shown as a separate line item on the invoice.

---

## Best Practices

- **Use descriptive codes.** Codes like `SUMMER20` or `LAUNCH50OFF` are easier for customers and support teams to understand than random strings.
- **Set max redemptions** for promotional campaigns to control costs.
- **Monitor `times_redeemed`** to track campaign performance.
- **Deactivate expired coupons** by setting `active` to `false` to prevent further use.
