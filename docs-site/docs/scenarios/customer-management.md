---
sidebar_position: 3
---

# Customer Management

Customers represent the businesses or individuals who purchase your plans. This guide covers creating, listing, and updating customers, including their payment methods and tax information.

## The Customer Object

```json
{
  "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
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
  "gstin": null,
  "tax_type": null,
  "payment_method": {
    "type": "card",
    "last4": "4242",
    "brand": "visa",
    "exp_month": 12,
    "exp_year": 2028
  },
  "created_at": "2026-03-10T08:15:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique customer identifier. |
| `name` | string | Full name or business name. |
| `email` | string | Primary billing email address. |
| `billing_address` | object | Postal address used on invoices. |
| `gstin` | string | GST Identification Number (for Indian tax compliance). |
| `tax_type` | string | Type of tax ID (e.g., `gst`, `vat`, `sales_tax`). |
| `payment_method` | object | The customer's active payment instrument. |
| `created_at` | string | ISO 8601 timestamp of when the customer was created. |

---

## Create a Customer

Register a new customer in Recurso.

**POST** `/v1/customers`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | The customer's full name or business name. |
| `email` | string | Yes | The billing email address. |
| `billing_address` | object | No | Address object with `line1`, `line2`, `city`, `state`, `country`, `zip`. |
| `gstin` | string | No | GST Identification Number. |
| `tax_type` | string | No | Tax identifier type: `gst`, `vat`, or `sales_tax`. |

### Example: Basic Customer

```bash
curl -X POST https://api.recurso.dev/v1/customers \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp",
    "email": "billing@acme.com"
  }'
```

Response:
```json
{
  "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "name": "Acme Corp",
  "email": "billing@acme.com",
  "billing_address": null,
  "gstin": null,
  "tax_type": null,
  "payment_method": null,
  "created_at": "2026-06-23T14:00:00Z"
}
```

### Example: Customer with Full Details

```bash
curl -X POST https://api.recurso.dev/v1/customers \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TechStart India Pvt Ltd",
    "email": "accounts@techstart.in",
    "billing_address": {
      "line1": "12 MG Road",
      "line2": "Indiranagar",
      "city": "Bengaluru",
      "state": "KA",
      "country": "IN",
      "zip": "560038"
    },
    "gstin": "29ABCDE1234F1Z5",
    "tax_type": "gst"
  }'
```

Response:
```json
{
  "id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "name": "TechStart India Pvt Ltd",
  "email": "accounts@techstart.in",
  "billing_address": {
    "line1": "12 MG Road",
    "line2": "Indiranagar",
    "city": "Bengaluru",
    "state": "KA",
    "country": "IN",
    "zip": "560038"
  },
  "gstin": "29ABCDE1234F1Z5",
  "tax_type": "gst",
  "payment_method": null,
  "created_at": "2026-06-23T14:05:00Z"
}
```

---

## List Customers

Retrieve all customers for your tenant.

**GET** `/v1/customers`

```bash
curl https://api.recurso.dev/v1/customers \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Acme Corp",
      "email": "billing@acme.com",
      "billing_address": null,
      "gstin": null,
      "tax_type": null,
      "payment_method": null,
      "created_at": "2026-06-23T14:00:00Z"
    },
    {
      "id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
      "name": "TechStart India Pvt Ltd",
      "email": "accounts@techstart.in",
      "billing_address": {
        "line1": "12 MG Road",
        "line2": "Indiranagar",
        "city": "Bengaluru",
        "state": "KA",
        "country": "IN",
        "zip": "560038"
      },
      "gstin": "29ABCDE1234F1Z5",
      "tax_type": "gst",
      "payment_method": null,
      "created_at": "2026-06-23T14:05:00Z"
    }
  ],
  "has_more": false
}
```

---

## Update Payment Method

Attach or update a customer's payment method. This is the payment instrument that will be charged when invoices are due.

**PUT** `/v1/customers/:id/payment-method`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `type` | string | Yes | Payment method type (e.g., `card`, `bank_account`, `upi`). |
| `token` | string | Yes | A tokenized payment method from your payment gateway. |

```bash
curl -X PUT https://api.recurso.dev/v1/customers/cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890/payment-method \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "card",
    "token": "tok_7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d"
  }'
```

Response:
```json
{
  "id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "name": "Acme Corp",
  "email": "billing@acme.com",
  "payment_method": {
    "type": "card",
    "last4": "4242",
    "brand": "visa",
    "exp_month": 12,
    "exp_year": 2028
  },
  "created_at": "2026-06-23T14:00:00Z"
}
```

---

## Best Practices

- **Always collect an email address.** Recurso sends invoice notifications and payment receipts to this address.
- **Set billing address early.** Tax calculations and invoice formatting depend on the customer's location.
- **Add a payment method before creating subscriptions.** While subscriptions can exist without a payment method, invoices will remain in the `open` state until one is provided.
- **Use tax fields for compliance.** If you operate in India, always collect `gstin` and set `tax_type` to `gst` so that invoices include the correct GST details.
