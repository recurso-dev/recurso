# Quickstart Guide

This guide will walk you through creating your first customer and subscription using the Recurso API.

## Prerequisites

- API Key (Get one from your [Developer Dashboard](/developers))
- Endpoint: `http://localhost:8080/v1` (Self-hosted) or `https://api.recurso.dev/v1`

## 1. Create a Plan

First, define what you are selling. Let's create a "Pro Plan" for $30/month.

```bash
curl -X POST https://api.recurso.dev/v1/plans \
  -u "recurso_sk_test_...:" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan",
    "code": "pro-monthly",
    "interval_unit": "month",
    "interval_count": 1,
    "amount": 3000,
    "currency": "USD"
  }'
```

Response:
```json
{
  "id": "plan_12345...",
  "name": "Pro Plan",
  "amount": 3000
}
```

## 2. Create a Customer

Next, register your user in Recurso.

```bash
curl -X POST https://api.recurso.dev/v1/customers \
  -u "recurso_sk_test_...:" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp",
    "email": "billing@acme.com"
  }'
```

## 3. Create a Subscription

Finally, subscribe the customer to the plan.

```bash
curl -X POST https://api.recurso.dev/v1/subscriptions \
  -u "recurso_sk_test_...:" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_...",
    "plan_id": "plan_..."
  }'
```

You're done! Recurso will now automatically generate invoices and charge the customer (if payment info is added) on a monthly basis.
