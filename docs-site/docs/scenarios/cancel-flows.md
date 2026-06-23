---
sidebar_position: 17
---

# Cancellation Flows & Retention

Cancel flows intercept cancellation requests with surveys, discount offers, and pause options to retain customers. Instead of letting a subscriber cancel immediately, you guide them through a configurable series of steps designed to understand their reason for leaving and present alternatives.

---

## Concepts

- **Flow** -- A named sequence of steps that a customer walks through when they attempt to cancel.
- **Step** -- An individual screen in the flow. Steps have a type: `survey` (ask why they are leaving), `offer` (present a discount or pause option), or `confirmation` (final cancellation confirmation).
- **Session** -- A single customer's journey through a flow. Sessions track which steps the customer has completed and their responses.

---

## Manage Flows

### List All Flows

```bash
curl -X GET https://api.recurso.dev/v1/cancel-flows \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "cflow_1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
      "name": "Standard Cancellation",
      "steps_count": 3,
      "sessions_started": 245,
      "retention_rate": 34.2,
      "created_at": "2026-02-15T10:00:00Z",
      "updated_at": "2026-06-01T08:30:00Z"
    }
  ],
  "has_more": false
}
```

### Create a Flow

```bash
curl -X POST https://api.recurso.dev/v1/cancel-flows \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan Cancellation",
    "steps": [
      {
        "type": "survey",
        "content": {
          "title": "We are sorry to see you go",
          "question": "What is the main reason you are cancelling?",
          "options": [
            "Too expensive",
            "Missing features",
            "Switching to a competitor",
            "No longer needed",
            "Other"
          ]
        }
      },
      {
        "type": "offer",
        "content": {
          "title": "Before you go, how about a discount?",
          "description": "We would like to offer you 30% off for the next 3 months.",
          "coupon_id": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
          "alternative_action": "pause",
          "pause_duration_days": 30
        }
      },
      {
        "type": "confirmation",
        "content": {
          "title": "Confirm cancellation",
          "message": "Your subscription will remain active until the end of your current billing period."
        }
      }
    ]
  }'
```

**Response:**

```json
{
  "id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "name": "Pro Plan Cancellation",
  "steps": [
    {
      "id": "cstep_a0b1c2d3-e4f5-6a7b-8c9d-0e1f2a3b4c5d",
      "type": "survey",
      "position": 1,
      "content": {
        "title": "We are sorry to see you go",
        "question": "What is the main reason you are cancelling?",
        "options": [
          "Too expensive",
          "Missing features",
          "Switching to a competitor",
          "No longer needed",
          "Other"
        ]
      }
    },
    {
      "id": "cstep_b1c2d3e4-f5a6-7b8c-9d0e-1f2a3b4c5d6e",
      "type": "offer",
      "position": 2,
      "content": {
        "title": "Before you go, how about a discount?",
        "description": "We would like to offer you 30% off for the next 3 months.",
        "coupon_id": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
        "alternative_action": "pause",
        "pause_duration_days": 30
      }
    },
    {
      "id": "cstep_c2d3e4f5-a6b7-8c9d-0e1f-2a3b4c5d6e7f",
      "type": "confirmation",
      "position": 3,
      "content": {
        "title": "Confirm cancellation",
        "message": "Your subscription will remain active until the end of your current billing period."
      }
    }
  ],
  "created_at": "2026-06-23T14:40:00Z",
  "updated_at": "2026-06-23T14:40:00Z"
}
```

### Get a Flow

```bash
curl -X GET https://api.recurso.dev/v1/cancel-flows/cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

### Update a Flow

```bash
curl -X PUT https://api.recurso.dev/v1/cancel-flows/cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan Cancellation v2"
  }'
```

**Response:**

```json
{
  "id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "name": "Pro Plan Cancellation v2",
  "steps_count": 3,
  "updated_at": "2026-06-23T14:45:00Z"
}
```

---

## Manage Steps

### Create a Step

Add a new step to an existing flow.

```bash
curl -X POST https://api.recurso.dev/v1/cancel-flows/cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b/steps \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "type": "offer",
    "content": {
      "title": "How about a free month?",
      "description": "We will extend your subscription by one month at no charge.",
      "coupon_id": "cpn_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
      "alternative_action": "pause",
      "pause_duration_days": 14
    }
  }'
```

**Response:**

```json
{
  "id": "cstep_d3e4f5a6-b7c8-9d0e-1f2a-3b4c5d6e7f8a",
  "flow_id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "type": "offer",
  "position": 4,
  "content": {
    "title": "How about a free month?",
    "description": "We will extend your subscription by one month at no charge.",
    "coupon_id": "cpn_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
    "alternative_action": "pause",
    "pause_duration_days": 14
  },
  "created_at": "2026-06-23T14:50:00Z"
}
```

### Update a Step

```bash
curl -X PUT https://api.recurso.dev/v1/cancel-flows/steps/cstep_d3e4f5a6-b7c8-9d0e-1f2a-3b4c5d6e7f8a \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "content": {
      "title": "Stay with us -- free month on us!",
      "description": "We will extend your subscription by one month at no charge.",
      "coupon_id": "cpn_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
      "alternative_action": "pause",
      "pause_duration_days": 14
    }
  }'
```

**Response:**

```json
{
  "id": "cstep_d3e4f5a6-b7c8-9d0e-1f2a-3b4c5d6e7f8a",
  "flow_id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "type": "offer",
  "position": 4,
  "content": {
    "title": "Stay with us -- free month on us!",
    "description": "We will extend your subscription by one month at no charge.",
    "coupon_id": "cpn_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
    "alternative_action": "pause",
    "pause_duration_days": 14
  },
  "updated_at": "2026-06-23T14:55:00Z"
}
```

### Delete a Step

```bash
curl -X DELETE https://api.recurso.dev/v1/cancel-flows/steps/cstep_d3e4f5a6-b7c8-9d0e-1f2a-3b4c5d6e7f8a \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "deleted": true,
  "id": "cstep_d3e4f5a6-b7c8-9d0e-1f2a-3b4c5d6e7f8a"
}
```

---

## Sessions

Sessions represent a customer's real-time journey through a cancellation flow. Start a session when a customer clicks "Cancel", then submit their response at each step.

### Start a Session

```bash
curl -X POST https://api.recurso.dev/v1/cancel-flows/sessions/start \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
    "flow_id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b"
  }'
```

**Response:**

```json
{
  "id": "csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "flow_id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "status": "in_progress",
  "current_step": {
    "id": "cstep_a0b1c2d3-e4f5-6a7b-8c9d-0e1f2a3b4c5d",
    "type": "survey",
    "position": 1,
    "content": {
      "title": "We are sorry to see you go",
      "question": "What is the main reason you are cancelling?",
      "options": [
        "Too expensive",
        "Missing features",
        "Switching to a competitor",
        "No longer needed",
        "Other"
      ]
    }
  },
  "started_at": "2026-06-23T15:00:00Z"
}
```

### Submit a Step Response

After the customer responds to the current step, submit their answer to advance to the next step.

```bash
curl -X POST https://api.recurso.dev/v1/cancel-flows/sessions/csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b/submit \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "response": {
      "selected_option": "Too expensive",
      "comment": "The price increased too much after the trial."
    }
  }'
```

**Response:**

```json
{
  "id": "csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b",
  "status": "in_progress",
  "completed_steps": 1,
  "total_steps": 3,
  "current_step": {
    "id": "cstep_b1c2d3e4-f5a6-7b8c-9d0e-1f2a3b4c5d6e",
    "type": "offer",
    "position": 2,
    "content": {
      "title": "Before you go, how about a discount?",
      "description": "We would like to offer you 30% off for the next 3 months.",
      "coupon_id": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
      "alternative_action": "pause",
      "pause_duration_days": 30
    }
  }
}
```

If the customer accepts the offer, submit the response with `"action": "accept_offer"` or `"action": "pause"`. If they decline, submit `"action": "decline"` to proceed to the next step.

```bash
curl -X POST https://api.recurso.dev/v1/cancel-flows/sessions/csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b/submit \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "response": {
      "action": "accept_offer"
    }
  }'
```

**Response (customer retained):**

```json
{
  "id": "csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b",
  "status": "retained",
  "outcome": "accepted_offer",
  "coupon_applied": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
  "completed_at": "2026-06-23T15:02:00Z"
}
```

### Get a Session

Retrieve full session details including all step responses.

```bash
curl -X GET https://api.recurso.dev/v1/cancel-flows/sessions/csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "csess_e4f5a6b7-c8d9-0e1f-2a3b-4c5d6e7f8a9b",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "flow_id": "cflow_7e8f9a0b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
  "status": "retained",
  "outcome": "accepted_offer",
  "responses": [
    {
      "step_id": "cstep_a0b1c2d3-e4f5-6a7b-8c9d-0e1f2a3b4c5d",
      "step_type": "survey",
      "response": {
        "selected_option": "Too expensive",
        "comment": "The price increased too much after the trial."
      },
      "submitted_at": "2026-06-23T15:01:00Z"
    },
    {
      "step_id": "cstep_b1c2d3e4-f5a6-7b8c-9d0e-1f2a3b4c5d6e",
      "step_type": "offer",
      "response": {
        "action": "accept_offer"
      },
      "submitted_at": "2026-06-23T15:02:00Z"
    }
  ],
  "started_at": "2026-06-23T15:00:00Z",
  "completed_at": "2026-06-23T15:02:00Z"
}
```

---

## Flow Statistics

Get aggregate statistics across all sessions for your cancel flows.

```bash
curl -X GET "https://api.recurso.dev/v1/cancel-flows/stats?from=2026-06-01&to=2026-06-23" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "period": {
    "from": "2026-06-01T00:00:00Z",
    "to": "2026-06-23T23:59:59Z"
  },
  "total_sessions": 142,
  "completed_sessions": 138,
  "outcomes": {
    "cancelled": 91,
    "retained": 34,
    "paused": 13
  },
  "retention_rate": 34.06,
  "top_cancel_reasons": [
    { "reason": "Too expensive", "count": 42, "percentage": 30.4 },
    { "reason": "Missing features", "count": 28, "percentage": 20.3 },
    { "reason": "Switching to a competitor", "count": 19, "percentage": 13.8 },
    { "reason": "No longer needed", "count": 35, "percentage": 25.4 },
    { "reason": "Other", "count": 14, "percentage": 10.1 }
  ],
  "offer_acceptance_rate": 24.6,
  "average_session_duration_seconds": 47
}
```

---

## Best Practices

1. **Start with a survey** -- Always ask why the customer is leaving. This data is invaluable even if you cannot save the customer.
2. **Personalize offers** -- Use churn score data (see [Churn Prevention](./churn-prevention)) to tailor the discount or pause duration.
3. **Keep flows short** -- Three steps (survey, offer, confirmation) is the sweet spot. More steps increase drop-off.
4. **A/B test flows** -- Create multiple flows and assign them to different plan tiers or customer segments to measure which retention strategies work best.
5. **Review stats weekly** -- Use `GET /v1/cancel-flows/stats` to track retention rates and adjust your offers based on the most common cancel reasons.
