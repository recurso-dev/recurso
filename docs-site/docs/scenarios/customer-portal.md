---
sidebar_position: 12
---

# Customer Portal

Recurso provides a self-service customer portal where your end customers can view their profile, download invoices, redeem gift codes, and manage their account -- all without contacting your support team. Authentication is handled via magic links, so customers never need a password.

---

## Magic Link Authentication

The portal uses a passwordless magic link flow. The customer receives a one-time login link via email, clicks it, and is authenticated with a session token.

### Step 1: Request a Magic Link

Send a magic link to the customer's email address.

**POST** `/portal/auth/request`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_email` | string | Yes | The email address of the customer requesting access. |

```bash
curl -X POST https://api.recurso.dev/portal/auth/request \
  -H "Content-Type: application/json" \
  -d '{
    "customer_email": "billing@acme.com"
  }'
```

Response:
```json
{
  "message": "Magic link sent successfully.",
  "email": "billing@acme.com",
  "expires_in": 600
}
```

The customer receives an email containing a link like:

```
https://api.recurso.dev/portal/auth/verify?token=mlk_a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0
```

The token expires after 10 minutes (600 seconds).

### Step 2: Verify the Token

When the customer clicks the magic link, verify the token to establish a session.

**GET** `/portal/auth/verify?token=...`

```bash
curl "https://api.recurso.dev/portal/auth/verify?token=mlk_a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0"
```

Response:
```json
{
  "session_token": "sess_d4c3b2a1-e5f6-7890-abcd-1234567890ab",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "email": "billing@acme.com",
  "expires_at": "2026-06-24T14:00:00Z"
}
```

Use the `session_token` as a Bearer token for all subsequent portal API calls. Sessions are valid for 24 hours.

---

## Authenticated Portal Endpoints

All portal API endpoints require the session token obtained from the magic link verification step.

### Get Customer Profile

Retrieve the authenticated customer's profile information.

**GET** `/portal/api/profile`

```bash
curl https://api.recurso.dev/portal/api/profile \
  -H "Authorization: Bearer sess_d4c3b2a1-e5f6-7890-abcd-1234567890ab"
```

Response:
```json
{
  "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "name": "Acme Corp",
  "email": "billing@acme.com",
  "billing_address": {
    "line1": "742 Evergreen Terrace",
    "line2": "Suite 400",
    "city": "San Francisco",
    "state": "CA",
    "country": "US",
    "zip": "94105"
  },
  "active_subscriptions": [
    {
      "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "plan_name": "Pro Plan",
      "status": "active",
      "current_period_end": "2026-07-23T00:00:00Z"
    }
  ],
  "created_at": "2026-03-10T08:15:00Z"
}
```

### Get Invoices

Retrieve all invoices for the authenticated customer.

**GET** `/portal/api/invoices`

```bash
curl https://api.recurso.dev/portal/api/invoices \
  -H "Authorization: Bearer sess_d4c3b2a1-e5f6-7890-abcd-1234567890ab"
```

Response:
```json
{
  "data": [
    {
      "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
      "invoice_number": "INV-2026-0042",
      "status": "paid",
      "total": 5782,
      "currency": "USD",
      "due_date": "2026-07-01T00:00:00Z",
      "paid_at": "2026-06-18T09:30:00Z",
      "pdf_url": "https://api.recurso.dev/v1/invoices/inv_3c4d5e6f-7a8b-9012-cdef-345678901234/pdf",
      "created_at": "2026-06-01T00:00:00Z"
    },
    {
      "id": "inv_ee11ff22-3344-5566-7788-99aabbccdd00",
      "invoice_number": "INV-2026-0041",
      "status": "paid",
      "total": 5782,
      "currency": "USD",
      "due_date": "2026-06-01T00:00:00Z",
      "paid_at": "2026-05-20T11:15:00Z",
      "pdf_url": "https://api.recurso.dev/v1/invoices/inv_ee11ff22-3344-5566-7788-99aabbccdd00/pdf",
      "created_at": "2026-05-01T00:00:00Z"
    }
  ],
  "has_more": false
}
```

### Redeem a Gift Code

Allow the authenticated customer to redeem a gift subscription code directly from the portal.

**POST** `/portal/api/redeem`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `code` | string | Yes | The gift subscription code to redeem. |

```bash
curl -X POST https://api.recurso.dev/portal/api/redeem \
  -H "Authorization: Bearer sess_d4c3b2a1-e5f6-7890-abcd-1234567890ab" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "GIFT-PRO-2H6J-R8T0"
  }'
```

Response:
```json
{
  "gift_id": "gift_b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e",
  "status": "redeemed",
  "subscription_id": "sub_f6a7b8c9-d0e1-2345-fa0b-678901234567",
  "plan_name": "Pro Monthly",
  "subscription_starts_at": "2026-06-23T16:00:00Z",
  "subscription_ends_at": "2026-07-23T16:00:00Z",
  "redeemed_at": "2026-06-23T16:00:00Z"
}
```

### Logout

Invalidate the current portal session.

**POST** `/portal/api/logout`

```bash
curl -X POST https://api.recurso.dev/portal/api/logout \
  -H "Authorization: Bearer sess_d4c3b2a1-e5f6-7890-abcd-1234567890ab"
```

Response:
```json
{
  "message": "Session invalidated successfully."
}
```

---

## Read-Only Portal Data Endpoint

This server-side API endpoint lets your backend fetch portal-ready data for a specific customer. It uses your standard API key authentication (not the portal session token).

**GET** `/v1/portal/:tenant_id/:customer_id`

```bash
curl https://api.recurso.dev/v1/portal/ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321/cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "customer": {
    "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "name": "Acme Corp",
    "email": "billing@acme.com"
  },
  "subscriptions": [
    {
      "id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "plan_name": "Pro Plan",
      "status": "active",
      "current_period_start": "2026-06-23T00:00:00Z",
      "current_period_end": "2026-07-23T00:00:00Z"
    }
  ],
  "invoices": [
    {
      "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
      "invoice_number": "INV-2026-0042",
      "status": "paid",
      "total": 5782,
      "currency": "USD",
      "paid_at": "2026-06-18T09:30:00Z"
    }
  ],
  "payment_method": {
    "type": "card",
    "last4": "4242",
    "brand": "visa",
    "exp_month": 12,
    "exp_year": 2028
  }
}
```

This endpoint is useful for pre-rendering portal data on your server or building a custom portal UI.

---

## Dashboard Page

Recurso hosts a ready-made dashboard page for each customer. You can embed this in an iframe or redirect your customers directly.

**GET** `/portal/:customer_id`

```
https://api.recurso.dev/portal/cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890
```

This renders a full HTML page showing the customer's active subscriptions, recent invoices, payment method on file, and gift redemption form. The customer must be authenticated via a magic link session to view the page.

If the customer is not authenticated, they are redirected to the magic link request flow.

---

## End-to-End Portal Flow

Here is the typical self-service portal experience from the customer's perspective:

1. **Customer visits portal** -- They navigate to your portal link or click a "Manage Billing" button in your app.
2. **Enter email** -- The customer enters their billing email. A magic link is sent.
3. **Click magic link** -- The customer clicks the link in their email. A session is established.
4. **View dashboard** -- The portal dashboard loads showing active subscriptions, invoices, and payment details.
5. **Download invoices** -- The customer can view or download PDF invoices for their records.
6. **Redeem gifts** -- If the customer has a gift code, they can enter it to activate a gifted subscription.
7. **Logout** -- The customer ends their session when finished.

---

## Best Practices

- **Use the hosted dashboard for quick setup.** The `/portal/:customer_id` page gives your customers a complete self-service experience with no frontend work required.
- **Use the read-only API for custom portals.** If you need full control over the UI, use the `/v1/portal/:tenant_id/:customer_id` endpoint to fetch data and render it in your own application.
- **Magic link tokens are single-use.** Once verified, the token is consumed. The session token is what persists for the duration of the session.
- **Session tokens expire after 24 hours.** Prompt customers to re-authenticate if their session has expired.
