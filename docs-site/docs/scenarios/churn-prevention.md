---
sidebar_position: 16
---

# Churn Prevention & Scoring

Recurso includes an ML-based churn prediction engine that scores every active customer based on behavioral signals. Use these scores to identify at-risk accounts early and take proactive retention actions before the customer cancels.

---

## How Churn Scoring Works

Recurso's churn model evaluates each customer on a rolling basis using the following factors:

| Factor | Weight | Description |
|---|---|---|
| Payment failures | High | Repeated failed charges or expired cards signal disengagement. |
| Usage decline | High | Decreasing API calls, logins, or metered usage over the last 30/60/90 days. |
| Support tickets | Medium | Spike in support ticket volume or negative sentiment in tickets. |
| Downgrade history | Medium | Recent plan downgrades or removal of add-ons. |
| Engagement drop | Medium | Fewer dashboard visits, email opens, or portal logins. |
| Invoice disputes | Low | Credit note requests or chargeback history. |
| Tenure | Low | Newer customers have a higher baseline churn probability. |

The model produces a **churn score** between 0 and 100. Scores above 70 are classified as **high risk**, scores between 40-70 as **medium risk**, and scores below 40 as **low risk**.

---

## Get Churn Score for a Customer

Retrieve the churn risk score and contributing factors for a specific customer.

```bash
curl -X GET https://api.recurso.dev/v1/customers/cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890/churn \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "score": 78,
  "risk_level": "high",
  "factors": [
    {
      "name": "payment_failures",
      "impact": "high",
      "details": "3 failed payment attempts in the last 30 days"
    },
    {
      "name": "usage_decline",
      "impact": "high",
      "details": "API usage dropped 62% compared to 60-day average"
    },
    {
      "name": "support_tickets",
      "impact": "medium",
      "details": "2 open tickets with negative sentiment"
    }
  ],
  "recommended_actions": [
    "Reach out with a personalized retention offer",
    "Resolve open support tickets urgently",
    "Offer a temporary discount or plan pause"
  ],
  "scored_at": "2026-06-23T06:00:00Z",
  "next_score_at": "2026-06-24T06:00:00Z"
}
```

---

## List High-Risk Customers

Fetch all customers with a churn score above the high-risk threshold. Results are sorted by score descending.

```bash
curl -X GET "https://api.recurso.dev/v1/churn/high-risk?limit=20&offset=0" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Acme Corp",
      "customer_email": "billing@acme.com",
      "score": 78,
      "risk_level": "high",
      "mrr": 4999900,
      "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
      "plan_name": "Enterprise Annual",
      "scored_at": "2026-06-23T06:00:00Z"
    },
    {
      "customer_id": "cust_b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "customer_name": "Globex Industries",
      "customer_email": "accounts@globex.io",
      "score": 74,
      "risk_level": "high",
      "mrr": 1999900,
      "subscription_id": "sub_e1d2c3b4-a5f6-7890-fedc-ba9876543210",
      "plan_name": "Growth Monthly",
      "scored_at": "2026-06-23T06:00:00Z"
    }
  ],
  "total": 12,
  "has_more": false
}
```

---

## List Churn Alerts

Churn alerts are automatically generated when a customer's score crosses a threshold or when a significant risk event occurs (e.g., third consecutive payment failure). Alerts remain active until acknowledged.

```bash
curl -X GET "https://api.recurso.dev/v1/churn/alerts?status=active&limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "churn_alert_9a8b7c6d-5e4f-3a2b-1c0d-e9f8a7b6c5d4",
      "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Acme Corp",
      "alert_type": "score_threshold_crossed",
      "message": "Churn score increased from 58 to 78 in 7 days",
      "score": 78,
      "previous_score": 58,
      "risk_level": "high",
      "status": "active",
      "created_at": "2026-06-23T06:05:00Z"
    },
    {
      "id": "churn_alert_8b7c6d5e-4f3a-2b1c-0de9-f8a7b6c5d4e3",
      "customer_id": "cust_b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "customer_name": "Globex Industries",
      "alert_type": "payment_failure_streak",
      "message": "3 consecutive payment failures in the last 14 days",
      "score": 74,
      "previous_score": 74,
      "risk_level": "high",
      "status": "active",
      "created_at": "2026-06-22T12:00:00Z"
    }
  ],
  "total": 5,
  "has_more": false
}
```

---

## Acknowledge a Churn Alert

Mark a churn alert as acknowledged once your team has reviewed it and taken action. This removes it from the active alerts list.

```bash
curl -X POST https://api.recurso.dev/v1/churn/alerts/churn_alert_9a8b7c6d-5e4f-3a2b-1c0d-e9f8a7b6c5d4/ack \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "churn_alert_9a8b7c6d-5e4f-3a2b-1c0d-e9f8a7b6c5d4",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "acknowledged",
  "acknowledged_at": "2026-06-23T14:35:00Z"
}
```

---

## Recommended Workflow

1. **Set up a daily review** -- Query `GET /v1/churn/high-risk` each morning to identify at-risk customers.
2. **Subscribe to webhook events** -- Listen for `churn.alert.created` events to get real-time notifications when new alerts fire.
3. **Acknowledge and act** -- For each alert, investigate the contributing factors, reach out to the customer, and acknowledge the alert once action is taken.
4. **Track outcomes** -- Monitor whether the customer's score improves after your intervention by re-checking `GET /v1/customers/:id/churn` over the following weeks.
