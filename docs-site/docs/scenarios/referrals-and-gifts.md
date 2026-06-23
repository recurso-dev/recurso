---
sidebar_position: 20
---

# Referrals & Gift Subscriptions

Recurso supports two growth-oriented features: **referral programs** that reward existing customers for bringing in new ones, and **gift subscriptions** that allow anyone to purchase a subscription on behalf of someone else.

---

## Referrals

### How Referrals Work

1. **Create a referral** for an existing customer. This generates a unique referral code.
2. **Share the code** with prospective customers (via email, in-app share, social media).
3. **New customer signs up** and applies the referral code during checkout.
4. **Qualify the referral** once the new customer completes their first payment. This triggers rewards for both the referrer and the referee.

Referral rewards are configurable per program. Common setups include:
- Referrer gets a credit or coupon applied to their next invoice.
- Referee gets a discount on their first billing cycle.

### List Referrals

```bash
curl -X GET "https://api.recurso.dev/v1/referrals?limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "ref_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "referrer_name": "Acme Corp",
      "code": "ACME-FRIEND-7X2K",
      "status": "qualified",
      "referee_customer_id": "cust_c3d4e5f6-a7b8-9012-cdef-345678901234",
      "referee_name": "Initech LLC",
      "referrer_reward": {
        "type": "credit",
        "amount": 50000,
        "currency": "INR"
      },
      "referee_reward": {
        "type": "coupon",
        "coupon_id": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
        "description": "20% off first month"
      },
      "qualified_at": "2026-06-15T12:00:00Z",
      "created_at": "2026-06-01T10:00:00Z"
    },
    {
      "id": "ref_b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "referrer_name": "Acme Corp",
      "code": "ACME-FRIEND-9M4P",
      "status": "pending",
      "referee_customer_id": null,
      "referee_name": null,
      "referrer_reward": null,
      "referee_reward": null,
      "qualified_at": null,
      "created_at": "2026-06-20T08:00:00Z"
    }
  ],
  "has_more": false
}
```

### Create a Referral

Create a new referral entry for an existing customer. A referral code is auto-generated.

```bash
curl -X POST https://api.recurso.dev/v1/referrals \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }'
```

**Response:**

```json
{
  "id": "ref_c3d4e5f6-a7b8-9012-cdef-345678901234",
  "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "code": "ACME-FRIEND-3J8W",
  "status": "pending",
  "created_at": "2026-06-23T15:40:00Z"
}
```

### Generate a Custom Referral Code

Generate a branded or custom referral code for a specific campaign.

```bash
curl -X POST https://api.recurso.dev/v1/referrals/generate-code \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "prefix": "SUMMER26",
    "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }'
```

**Response:**

```json
{
  "code": "SUMMER26-ACME-5R9T",
  "referral_id": "ref_d4e5f6a7-b8c9-0123-def0-456789012345",
  "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-06-23T15:42:00Z"
}
```

### Qualify a Referral

Once the referred customer completes their first payment, qualify the referral to trigger rewards for both parties.

```bash
curl -X POST https://api.recurso.dev/v1/referrals/ref_c3d4e5f6-a7b8-9012-cdef-345678901234/qualify \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "referee_customer_id": "cust_d4e5f6a7-b8c9-0123-def0-456789012345"
  }'
```

**Response:**

```json
{
  "id": "ref_c3d4e5f6-a7b8-9012-cdef-345678901234",
  "status": "qualified",
  "referrer_customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "referee_customer_id": "cust_d4e5f6a7-b8c9-0123-def0-456789012345",
  "referrer_reward": {
    "type": "credit",
    "amount": 50000,
    "currency": "INR",
    "applied_to_invoice": "inv_e5f6a7b8-c9d0-1234-ef01-567890123456"
  },
  "referee_reward": {
    "type": "coupon",
    "coupon_id": "cpn_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
    "description": "20% off first month"
  },
  "qualified_at": "2026-06-23T15:45:00Z"
}
```

---

## Referral Statuses

| Status | Description |
|---|---|
| `pending` | Referral code created but not yet used by a referee. |
| `claimed` | A new customer signed up using the referral code but has not paid yet. |
| `qualified` | The referee completed their first payment; rewards have been distributed. |
| `expired` | The referral code expired before being used (default: 90 days). |
| `revoked` | The referral was manually revoked (e.g., fraud detected). |

---

## Gift Subscriptions

Gift subscriptions let anyone purchase a subscription plan and send it to a recipient. The recipient redeems the gift with a unique code and gets immediate access without entering payment details.

### How Gift Subscriptions Work

1. **Purchase a gift** -- The sender selects a plan, pays for it, and provides the recipient's email.
2. **Recipient receives an email** with a unique gift code and redemption link.
3. **Recipient redeems the code** -- They create an account (or log in) and apply the gift code. A subscription is created automatically with the gifted plan and duration.

### List Gifts

```bash
curl -X GET "https://api.recurso.dev/v1/gifts?limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "gift_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
      "plan_id": "plan_3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d",
      "plan_name": "Pro Monthly",
      "sender_email": "alice@example.com",
      "recipient_email": "bob@example.com",
      "code": "GIFT-PRO-8K3M-X2Y4",
      "status": "redeemed",
      "amount": 199900,
      "currency": "INR",
      "redeemed_by": "cust_e5f6a7b8-c9d0-1234-ef01-567890123456",
      "redeemed_at": "2026-06-18T09:00:00Z",
      "purchased_at": "2026-06-10T14:30:00Z"
    },
    {
      "id": "gift_a7b8c9d0-e1f2-3a4b-5c6d-7e8f9a0b1c2d",
      "plan_id": "plan_3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d",
      "plan_name": "Pro Monthly",
      "sender_email": "carol@example.com",
      "recipient_email": "dave@example.com",
      "code": "GIFT-PRO-5N7Q-W1Z3",
      "status": "purchased",
      "amount": 199900,
      "currency": "INR",
      "redeemed_by": null,
      "redeemed_at": null,
      "purchased_at": "2026-06-22T16:00:00Z"
    }
  ],
  "has_more": false
}
```

### Purchase a Gift Subscription

```bash
curl -X POST https://api.recurso.dev/v1/gifts/purchase \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "plan_3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d",
    "sender_email": "alice@example.com",
    "recipient_email": "bob@example.com",
    "sender_name": "Alice Johnson",
    "message": "Happy birthday! Enjoy a year of Pro features on me."
  }'
```

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `plan_id` | string | Yes | The plan to gift. |
| `sender_email` | string | Yes | The email address of the person purchasing the gift. |
| `recipient_email` | string | Yes | The email address of the gift recipient. |
| `sender_name` | string | No | Display name of the sender, shown in the gift email. |
| `message` | string | No | A personal message included in the gift email. |

**Response:**

```json
{
  "id": "gift_b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e",
  "plan_id": "plan_3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d",
  "plan_name": "Pro Monthly",
  "sender_email": "alice@example.com",
  "recipient_email": "bob@example.com",
  "code": "GIFT-PRO-2H6J-R8T0",
  "status": "purchased",
  "amount": 199900,
  "currency": "INR",
  "checkout_url": "https://billing.recurso.dev/gifts/GIFT-PRO-2H6J-R8T0/checkout",
  "purchased_at": "2026-06-23T15:50:00Z"
}
```

The sender is redirected to `checkout_url` to complete payment. Once paid, the recipient receives an email with the gift code and a redemption link.

### Redeem a Gift Subscription

The recipient redeems the gift code to activate their subscription.

```bash
curl -X POST https://api.recurso.dev/v1/gifts/redeem \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "code": "GIFT-PRO-2H6J-R8T0",
    "customer_id": "cust_e5f6a7b8-c9d0-1234-ef01-567890123456"
  }'
```

**Response:**

```json
{
  "id": "gift_b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e",
  "status": "redeemed",
  "redeemed_by": "cust_e5f6a7b8-c9d0-1234-ef01-567890123456",
  "subscription_id": "sub_f6a7b8c9-d0e1-2345-fa0b-678901234567",
  "plan_name": "Pro Monthly",
  "subscription_starts_at": "2026-06-23T15:55:00Z",
  "subscription_ends_at": "2026-07-23T15:55:00Z",
  "redeemed_at": "2026-06-23T15:55:00Z"
}
```

Redemption creates a new subscription for the recipient. The subscription duration matches the gifted plan's billing interval. No payment method is required from the recipient for the gifted period.

---

## Gift Statuses

| Status | Description |
|---|---|
| `purchased` | Gift has been paid for; awaiting recipient redemption. |
| `redeemed` | Recipient has activated the gift subscription. |
| `expired` | Gift code was not redeemed within 365 days of purchase. |
| `refunded` | The gift purchase was refunded before redemption. |
