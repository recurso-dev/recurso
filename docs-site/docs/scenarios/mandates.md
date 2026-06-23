---
sidebar_position: 18
---

# UPI Mandates

UPI mandates enable recurring payments in India by allowing customers to authorize automatic debits from their UPI-linked bank account. Recurso manages the full mandate lifecycle -- creation, customer authorization, auto-debit execution on billing dates, and revocation.

---

## How UPI Mandates Work

1. **Create a mandate** -- You create a mandate with the customer's UPI ID, the maximum debit amount, and the billing frequency.
2. **Customer authorization** -- The customer receives an authorization request on their UPI app (Google Pay, PhonePe, Paytm, etc.) and approves the mandate.
3. **Mandate becomes active** -- Once approved, the mandate status changes to `active` and Recurso stores it against the customer and subscription.
4. **Auto-debit on billing date** -- On each billing cycle, Recurso initiates a UPI auto-debit against the active mandate. A pre-debit notification is sent 24 hours in advance per RBI guidelines.
5. **Revocation** -- Either the customer or the merchant can revoke the mandate at any time, stopping future auto-debits.

---

## Create a Mandate

```bash
curl -X POST https://api.recurso.dev/v1/mandates \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "amount": 99900,
    "frequency": "monthly",
    "upi_id": "customer@okaxis",
    "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
    "start_date": "2026-07-01",
    "end_date": "2027-07-01",
    "description": "Pro Plan monthly subscription"
  }'
```

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The customer authorizing the mandate. |
| `amount` | integer | Yes | Maximum debit amount per cycle in paise (e.g., 99900 = INR 999.00). |
| `frequency` | string | Yes | Billing frequency: `weekly`, `monthly`, `quarterly`, `yearly`. |
| `upi_id` | string | Yes | The customer's UPI ID (e.g., `customer@okaxis`). |
| `subscription_id` | string | No | Link the mandate to a specific subscription for auto-debit. |
| `start_date` | string | No | Mandate start date (ISO 8601 date). |
| `end_date` | string | No | Mandate expiry date (ISO 8601 date). |
| `description` | string | No | Human-readable description shown in the customer's UPI app. |

**Response:**

```json
{
  "id": "mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "amount": 99900,
  "frequency": "monthly",
  "upi_id": "customer@okaxis",
  "status": "pending_authorization",
  "authorization_url": "upi://mandate?pa=recurso@ybl&pn=Recurso&tid=TXN1234567890&tr=mdt_c3d4e5f6&am=999.00&cu=INR&mode=00",
  "start_date": "2026-07-01",
  "end_date": "2027-07-01",
  "description": "Pro Plan monthly subscription",
  "created_at": "2026-06-23T15:10:00Z"
}
```

After creation, the mandate is in `pending_authorization` status. The customer must approve it via their UPI app. Recurso sends the authorization request automatically and fires a `mandate.authorized` webhook when the customer approves.

---

## List Mandates

```bash
curl -X GET "https://api.recurso.dev/v1/mandates?customer_id=cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890&limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
      "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
      "amount": 99900,
      "frequency": "monthly",
      "upi_id": "customer@okaxis",
      "status": "active",
      "start_date": "2026-07-01",
      "end_date": "2027-07-01",
      "last_debit_at": "2026-06-01T00:05:00Z",
      "next_debit_at": "2026-07-01T00:05:00Z",
      "created_at": "2026-06-23T15:10:00Z"
    }
  ],
  "has_more": false
}
```

You can filter by `customer_id`, `subscription_id`, or `status` (`pending_authorization`, `active`, `paused`, `revoked`, `expired`).

---

## Retrieve a Mandate

```bash
curl -X GET https://api.recurso.dev/v1/mandates/mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "amount": 99900,
  "frequency": "monthly",
  "upi_id": "customer@okaxis",
  "status": "active",
  "authorization": {
    "authorized_at": "2026-06-23T15:15:00Z",
    "umn": "UMN1234567890abcdef",
    "ref_id": "REF0987654321"
  },
  "start_date": "2026-07-01",
  "end_date": "2027-07-01",
  "description": "Pro Plan monthly subscription",
  "last_debit_at": null,
  "next_debit_at": "2026-07-01T00:05:00Z",
  "debits_count": 0,
  "created_at": "2026-06-23T15:10:00Z"
}
```

---

## Revoke a Mandate

Revoke an active mandate to stop all future auto-debits. This can be triggered by the merchant or by the customer (via the customer portal or UPI app).

```bash
curl -X POST https://api.recurso.dev/v1/mandates/mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f/revoke \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "customer_requested"
  }'
```

**Response:**

```json
{
  "id": "mdt_c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
  "status": "revoked",
  "revoked_at": "2026-06-23T15:20:00Z",
  "reason": "customer_requested"
}
```

After revocation, the linked subscription is **not** automatically cancelled. It moves to a `payment_method_required` state, and the customer must add a new payment method or mandate to continue.

---

## Mandate Lifecycle Statuses

| Status | Description |
|---|---|
| `pending_authorization` | Mandate created, waiting for customer approval on UPI app. |
| `active` | Customer has authorized the mandate; auto-debits will execute. |
| `paused` | Mandate temporarily paused (e.g., subscription is paused). |
| `revoked` | Mandate revoked by merchant or customer; no further debits. |
| `expired` | Mandate has passed its `end_date`. |
| `failed` | Authorization was rejected by the customer or their bank. |

---

## Webhook Events

| Event | Description |
|---|---|
| `mandate.created` | A new mandate was created and is pending authorization. |
| `mandate.authorized` | The customer approved the mandate on their UPI app. |
| `mandate.authorization_failed` | The customer declined or the bank rejected the authorization. |
| `mandate.debit_initiated` | An auto-debit was initiated against the mandate. |
| `mandate.debit_success` | The auto-debit completed successfully. |
| `mandate.debit_failed` | The auto-debit failed (insufficient funds, technical error). |
| `mandate.revoked` | The mandate was revoked. |
| `mandate.expired` | The mandate reached its end date. |
