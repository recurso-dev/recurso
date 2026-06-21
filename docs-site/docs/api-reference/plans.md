# Plans

Plans define the billing frequency, pricing, and currency of your subscriptions.

## The Plan Object

```json
{
  "id": "plan_789",
  "name": "Gold Monthly",
  "code": "gold-monthly",
  "amount": 5000,
  "currency": "USD",
  "interval_unit": "month",
  "interval_count": 1
}
```

## Create a Plan

**POST** `/plans`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name of the plan. |
| `code` | string | No | Unique identifier for referencing in code. |
| `amount` | integer | Yes | Amount in cents (e.g. 1000 = $10.00). |
| `currency` | string | Yes | 3-letter ISO currency code. |
| `interval_unit` | string | Yes | `day`, `week`, `month`, or `year`. |
| `interval_count` | integer | Yes | Number of intervals (e.g. 1 for monthly, 3 for quarterly). |

## List Plans

**GET** `/plans`

Returns a list of all active plans.
