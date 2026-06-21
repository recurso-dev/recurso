# Subscriptions

Subscriptions allow you to charge a customer on a recurring basis.

## The Subscription Object

```json
{
  "id": "sub_456",
  "customer_id": "cust_123",
  "plan_id": "plan_789",
  "status": "active",
  "current_period_start": "2024-01-01T00:00:00Z",
  "current_period_end": "2024-02-01T00:00:00Z"
}
```

## Create a Subscription

**POST** `/subscriptions`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The ID of the customer to subscribe. |
| `plan_id` | string | Yes | The ID of the plan to subscribe to. |
| `start_date` | string | No | Anchor date for the subscription (ISO format). |

## Retrieve a Subscription

**GET** `/subscriptions/{id}`

Retrieves a subscription by its ID.
