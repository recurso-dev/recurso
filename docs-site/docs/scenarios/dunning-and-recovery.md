---
sidebar_position: 8
---

# Dunning & Payment Recovery

When a payment fails, Recurso automatically initiates a dunning process to recover the revenue. This guide covers how to configure multi-channel dunning campaigns, monitor their effectiveness, and leverage the smart retry system that uses bandit-based optimization to choose the best retry timing.

## How Dunning Works

1. A payment fails (e.g., expired card, insufficient funds).
2. Recurso fires a `payment_failed` or `invoice_overdue` trigger event.
3. The dunning campaign worker picks up the event and begins executing the campaign steps in order -- sending emails, SMS messages, or activating a payment wall.
4. In parallel, the smart retry system selects the optimal retry interval based on historical success rates for similar payment contexts.
5. If the customer pays, the campaign execution is marked as `recovered`. If all steps are exhausted without payment, it is marked as `exhausted`.

## Dunning Campaigns

A dunning campaign is a sequence of steps that Recurso executes when a trigger event occurs. Each step defines a communication channel (`email`, `sms`, or `in_app`), a delay, and a message template.

### Create a Campaign

**POST** `/v1/dunning-campaigns`

```bash
curl -X POST https://api.recurso.dev/v1/dunning-campaigns \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Standard Payment Recovery",
    "trigger_event": "payment_failed"
  }'
```

Response (`201 Created`):

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "name": "Standard Payment Recovery",
  "is_active": true,
  "trigger_event": "payment_failed",
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:00Z"
}
```

| Body Parameter   | Type   | Required | Description                                             |
|------------------|--------|----------|---------------------------------------------------------|
| `name`           | string | Yes      | A human-readable name for the campaign.                 |
| `trigger_event`  | string | Yes      | The event that triggers this campaign: `payment_failed` or `invoice_overdue`. |

### List Campaigns

**GET** `/v1/dunning-campaigns`

```bash
curl https://api.recurso.dev/v1/dunning-campaigns \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
[
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "name": "Standard Payment Recovery",
    "is_active": true,
    "trigger_event": "payment_failed",
    "created_at": "2026-06-23T10:00:00Z",
    "updated_at": "2026-06-23T10:00:00Z"
  }
]
```

### Get a Campaign

**GET** `/v1/dunning-campaigns/:id`

```bash
curl https://api.recurso.dev/v1/dunning-campaigns/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "name": "Standard Payment Recovery",
  "is_active": true,
  "trigger_event": "payment_failed",
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:00Z",
  "steps": [
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "campaign_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "step_order": 1,
      "channel": "email",
      "delay_hours": 24,
      "template_name": "payment_failed_reminder",
      "subject": "Action required: your payment failed",
      "body": "",
      "is_payment_wall": false,
      "created_at": "2026-06-23T10:05:00Z"
    }
  ]
}
```

### Update a Campaign

**PUT** `/v1/dunning-campaigns/:id`

```bash
curl -X PUT https://api.recurso.dev/v1/dunning-campaigns/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Aggressive Payment Recovery",
    "is_active": true,
    "trigger_event": "payment_failed"
  }'
```

Response (`200 OK`):

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "name": "Aggressive Payment Recovery",
  "is_active": true,
  "trigger_event": "payment_failed",
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T11:30:00Z"
}
```

| Body Parameter   | Type    | Required | Description                                |
|------------------|---------|----------|--------------------------------------------|
| `name`           | string  | No       | Updated campaign name.                     |
| `is_active`      | boolean | No       | Enable or disable the campaign.            |
| `trigger_event`  | string  | No       | Updated trigger event.                     |

## Campaign Steps

Steps define what happens at each stage of the dunning sequence. They are executed in `step_order`, with a delay between each step.

### Add a Step

**POST** `/v1/dunning-campaigns/:id/steps`

```bash
curl -X POST https://api.recurso.dev/v1/dunning-campaigns/a1b2c3d4-e5f6-7890-abcd-ef1234567890/steps \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "step_order": 1,
    "channel": "email",
    "delay_hours": 24,
    "template_name": "payment_failed_reminder",
    "subject": "Action required: your payment failed",
    "body": "Hi {{.CustomerName}}, your payment of {{.Amount}} failed. Please update your card.",
    "is_payment_wall": false
  }'
```

Response (`201 Created`):

```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "campaign_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "step_order": 1,
  "channel": "email",
  "delay_hours": 24,
  "template_name": "payment_failed_reminder",
  "subject": "Action required: your payment failed",
  "body": "Hi {{.CustomerName}}, your payment of {{.Amount}} failed. Please update your card.",
  "is_payment_wall": false,
  "created_at": "2026-06-23T10:05:00Z"
}
```

| Body Parameter   | Type    | Required | Description                                                        |
|------------------|---------|----------|--------------------------------------------------------------------|
| `step_order`     | integer | Yes      | Execution order (1, 2, 3...).                                      |
| `channel`        | string  | Yes      | Communication channel: `email`, `sms`, or `in_app`.               |
| `delay_hours`    | integer | No       | Hours to wait after the previous step before executing this one.   |
| `template_name`  | string  | No       | Name of the message template to use.                               |
| `subject`        | string  | No       | Email subject line (for `email` channel).                          |
| `body`           | string  | No       | Message body. Supports Go template variables.                      |
| `is_payment_wall`| boolean | No       | If `true`, this step activates a payment wall on the customer portal. |

**Example: A 3-step campaign**

1. **Step 1** (email, delay 24h): Friendly reminder with a link to update payment method.
2. **Step 2** (sms, delay 72h): SMS nudge for customers who have not responded.
3. **Step 3** (in_app, delay 168h): Payment wall -- blocks portal access until payment is resolved.

### Update a Step

**PUT** `/v1/dunning-campaigns/steps/:id`

```bash
curl -X PUT https://api.recurso.dev/v1/dunning-campaigns/steps/b2c3d4e5-f6a7-8901-bcde-f12345678901 \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "delay_hours": 48,
    "subject": "Urgent: please update your payment method"
  }'
```

Response (`200 OK`):

```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "campaign_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "step_order": 1,
  "channel": "email",
  "delay_hours": 48,
  "subject": "Urgent: please update your payment method",
  "body": "Hi {{.CustomerName}}, your payment of {{.Amount}} failed. Please update your card.",
  "is_payment_wall": false,
  "created_at": "2026-06-23T10:05:00Z"
}
```

### Delete a Step

**DELETE** `/v1/dunning-campaigns/steps/:id`

```bash
curl -X DELETE https://api.recurso.dev/v1/dunning-campaigns/steps/b2c3d4e5-f6a7-8901-bcde-f12345678901 \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "status": "deleted"
}
```

## Payment Wall

A payment wall blocks a customer's portal access until their overdue invoice is resolved. When a campaign step with `is_payment_wall: true` is executed, the customer sees a payment prompt instead of their normal dashboard.

### Check Payment Wall Status

**GET** `/v1/invoices/:id/payment-wall`

```bash
curl https://api.recurso.dev/v1/invoices/c3d4e5f6-a7b8-9012-cdef-123456789012/payment-wall \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "invoice_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "payment_wall_active": true
}
```

## Dunning Analytics

Monitor the performance of your dunning process with built-in analytics endpoints.

### Overview

**GET** `/v1/analytics/dunning/overview`

Returns high-level metrics about dunning performance: total retries, success rate, and amount recovered.

```bash
curl https://api.recurso.dev/v1/analytics/dunning/overview \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "total_retries": 1247,
  "successful_retries": 891,
  "success_rate": 0.7145,
  "total_amount_recovered": 4523800,
  "active_dunning_invoices": 34
}
```

### Weights (Bandit Model)

**GET** `/v1/analytics/dunning/weights`

Returns the learned weights from the bandit-based smart retry system. Each entry shows the average success rate for a given action (retry interval) within a specific payment context.

```bash
curl https://api.recurso.dev/v1/analytics/dunning/weights \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "context_key": "USD:card_declined:card:medium:2:established",
      "action_id": "24h",
      "average_reward": 0.72,
      "sample_count": 318,
      "updated_at": "2026-06-22T18:00:00Z"
    },
    {
      "context_key": "USD:insufficient_funds:card:small:5:new",
      "action_id": "3d",
      "average_reward": 0.58,
      "sample_count": 145,
      "updated_at": "2026-06-22T18:00:00Z"
    }
  ]
}
```

### History

**GET** `/v1/analytics/dunning/history`

Returns individual retry attempts and their outcomes.

```bash
curl "https://api.recurso.dev/v1/analytics/dunning/history?limit=20" \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "id": "d4e5f6a7-b8c9-0123-def0-1234567890ab",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "invoice_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "context_key": "USD:card_declined:card:medium:2:established",
      "action_id": "24h",
      "retry_interval": 86400,
      "outcome": "success",
      "reward": 1.0,
      "created_at": "2026-06-22T14:30:00Z"
    },
    {
      "id": "e5f6a7b8-c9d0-1234-ef01-234567890abc",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "invoice_id": "a1b2c3d4-5678-9012-abcd-ef1234567890",
      "context_key": "USD:insufficient_funds:card:large:4:veteran",
      "action_id": "1h",
      "retry_interval": 3600,
      "outcome": "failure",
      "reward": 0.0,
      "created_at": "2026-06-22T12:15:00Z"
    }
  ]
}
```

| Query Parameter | Type    | Default | Description                      |
|-----------------|---------|---------|----------------------------------|
| `limit`         | integer | 50      | Maximum number of records to return. |

## Smart Retry System

Recurso uses a contextual bandit algorithm to decide _when_ to retry a failed payment. Instead of a fixed schedule, the system learns the optimal retry interval for different payment contexts.

### How it works

1. **Context features**: When a payment fails, Recurso captures the context -- currency, error code, payment method, amount bucket (`small`, `medium`, `large`, `enterprise`), day of week, and customer age (`new`, `established`, `veteran`).
2. **Available actions**: The system chooses among four retry intervals: `1h`, `24h`, `3d`, and `7d`.
3. **Exploration vs. exploitation**: The bandit balances trying new intervals (exploration) with using the historically best-performing interval (exploitation) for each context.
4. **Learning**: After each retry, the outcome (success or failure) is recorded as a reward. Over time, the model converges on the optimal retry timing for each payment context.

### Amount buckets

| Bucket       | Amount range    |
|--------------|-----------------|
| `small`      | Under $10       |
| `medium`     | $10 -- $99.99   |
| `large`      | $100 -- $999.99 |
| `enterprise` | $1,000+         |

### Customer age buckets

| Bucket        | Account age   |
|---------------|---------------|
| `new`         | Under 30 days |
| `established` | 30 days -- 1 year |
| `veteran`     | Over 1 year   |

You can set the bandit strategy via the `DUNNING_STRATEGY` environment variable. The system starts with a default exploration rate and narrows it as more data is collected.
