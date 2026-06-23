---
sidebar_position: 1
---

# Subscription Lifecycle

This guide walks through the complete lifecycle of a subscription in Recurso -- from creation through plan changes, pausing, cancellation, and reactivation.

## The Subscription Object

Every subscription contains the following fields:

```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
  "status": "active",
  "current_period_start": "2026-06-01T00:00:00Z",
  "current_period_end": "2026-07-01T00:00:00Z",
  "cancel_at_period_end": false,
  "billing_anchor_type": "start_of_period",
  "payment_terms": "due_on_receipt",
  "coupon_id": null,
  "created_at": "2026-06-01T10:30:00Z"
}
```

### Status Values

| Status | Description |
|---|---|
| `active` | The subscription is in good standing and billing normally. |
| `trialing` | The subscription is within its trial period. No charges yet. |
| `paused` | Billing has been temporarily suspended. |
| `past_due` | An invoice is overdue but the subscription has not been canceled. |
| `unpaid` | Multiple invoices are overdue. Requires attention. |
| `canceled` | The subscription has been terminated. |

---

## Create a Subscription

Subscribe a customer to a plan.

**POST** `/v1/subscriptions`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The ID of the customer. |
| `plan_id` | string | Yes | The ID of the plan to subscribe to. |
| `start_date` | string | No | When the subscription should start (ISO 8601). Defaults to now. |
| `coupon_code` | string | No | A coupon code to apply a discount. |
| `billing_anchor_type` | string | No | Controls billing date alignment. `start_of_period` (default) or `start_of_month`. |
| `payment_terms` | string | No | When payment is due. One of `due_on_receipt`, `net15`, `net30`, `net60`. |

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
    "start_date": "2026-07-01T00:00:00Z",
    "coupon_code": "SUMMER20",
    "billing_anchor_type": "start_of_period",
    "payment_terms": "net30"
  }'
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
  "status": "active",
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "cancel_at_period_end": false,
  "billing_anchor_type": "start_of_period",
  "payment_terms": "net30",
  "coupon_id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## List Subscriptions

Retrieve all subscriptions for your tenant.

**GET** `/v1/subscriptions`

```bash
curl https://api.recurso.dev/v1/subscriptions \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "plan_id": "plan_9f8e7d6c-5b4a-3210-fedc-ba0987654321",
      "status": "active",
      "current_period_start": "2026-07-01T00:00:00Z",
      "current_period_end": "2026-08-01T00:00:00Z",
      "cancel_at_period_end": false,
      "billing_anchor_type": "start_of_period",
      "payment_terms": "net30",
      "coupon_id": "cpn_d4e5f6a7-b8c9-0123-4567-890abcdef123",
      "created_at": "2026-06-23T14:22:00Z"
    }
  ],
  "has_more": false
}
```

---

## Update / Change Plan

Switch a subscription to a different plan. When the plan changes mid-cycle, Recurso automatically calculates **proration**: the customer receives a credit for the unused portion of the current plan and is charged the prorated amount of the new plan for the remainder of the billing period.

**PUT** `/v1/subscriptions/:id`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `plan_id` | string | Yes | The ID of the new plan. |

```bash
curl -X PUT https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "plan_bb1122cc-dd33-4455-ee66-ff7788990011"
  }'
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "plan_id": "plan_bb1122cc-dd33-4455-ee66-ff7788990011",
  "status": "active",
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "cancel_at_period_end": false,
  "billing_anchor_type": "start_of_period",
  "payment_terms": "net30",
  "coupon_id": null,
  "created_at": "2026-06-23T14:22:00Z"
}
```

### How Proration Works

When a customer upgrades or downgrades mid-cycle, Recurso:

1. **Credits** the unused time on the old plan. For example, if a customer on a $30/month plan upgrades on day 15, they receive a ~$15 credit.
2. **Charges** the prorated cost of the new plan for the remaining days in the period.
3. **Adjusts** the next invoice to reflect the net difference.

The proration is applied automatically. No additional API calls are needed.

---

## Pause a Subscription

Temporarily suspend billing. While paused, no invoices are generated and the customer retains access until the current period ends (depending on your application logic).

**POST** `/v1/subscriptions/:id/pause`

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/pause \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "paused",
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "cancel_at_period_end": false,
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## Resume a Subscription

Resume a paused subscription. Billing resumes from the next period.

**POST** `/v1/subscriptions/:id/resume`

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/resume \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "active",
  "current_period_start": "2026-08-01T00:00:00Z",
  "current_period_end": "2026-09-01T00:00:00Z",
  "cancel_at_period_end": false,
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## Cancel a Subscription

Cancel a subscription either immediately or at the end of the current billing period.

**POST** `/v1/subscriptions/:id/cancel`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `immediately` | boolean | No | If `true`, cancels right now. If `false` (default), cancels at period end. |
| `reason` | string | No | Internal reason for cancellation. |
| `feedback` | string | No | Customer-facing feedback or note. |

### Cancel at End of Period

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/cancel \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "immediately": false,
    "reason": "customer_requested",
    "feedback": "Switching to a competitor product."
  }'
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "active",
  "cancel_at_period_end": true,
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "created_at": "2026-06-23T14:22:00Z"
}
```

When `immediately` is `false`, the subscription stays `active` until `current_period_end`, at which point it transitions to `canceled`.

### Cancel Immediately

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/cancel \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "immediately": true,
    "reason": "non_payment"
  }'
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "canceled",
  "cancel_at_period_end": false,
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## Reactivate a Subscription

Bring a canceled subscription back to life. This is only possible if the subscription was canceled at the end of the period and has not yet reached that date, or if your billing policy allows reactivation.

**POST** `/v1/subscriptions/:id/reactivate`

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/reactivate \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "active",
  "cancel_at_period_end": false,
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-01T00:00:00Z",
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## Extend a Subscription Period

Extend the current billing period without generating a new invoice. This is useful for granting complimentary time to a customer.

**POST** `/v1/subscriptions/:id/extend`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `days` | integer | Yes | Number of days to extend the current period by. |

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions/sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890/extend \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "days": 14
  }'
```

Response:
```json
{
  "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "active",
  "current_period_start": "2026-07-01T00:00:00Z",
  "current_period_end": "2026-08-15T00:00:00Z",
  "created_at": "2026-06-23T14:22:00Z"
}
```

---

## Lifecycle Summary

The typical subscription lifecycle flows like this:

```
  create --> active --> (pause) --> paused --> (resume) --> active
                 |                                           |
                 +--> (cancel) --> canceled --> (reactivate) -+
```

1. **Create** a subscription to start billing.
2. **Update** the plan to upgrade or downgrade (proration is automatic).
3. **Pause** to temporarily suspend billing.
4. **Resume** to restart billing after a pause.
5. **Cancel** to end the subscription immediately or at period end.
6. **Reactivate** to restore a canceled subscription.
7. **Extend** to grant additional time on the current period.
