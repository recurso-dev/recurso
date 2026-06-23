---
sidebar_position: 4
---

# Invoicing & Payments

Recurso automatically generates invoices for your subscriptions at the start of each billing period. This guide covers the invoice lifecycle, payment terms, PDF generation, and the customer checkout flow.

## The Invoice Object

```json
{
  "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "open",
  "subtotal": 4900,
  "tax_amount": 882,
  "total": 5782,
  "currency": "USD",
  "payment_terms": "net30",
  "due_date": "2026-08-01T00:00:00Z",
  "paid_at": null,
  "invoice_number": "INV-2026-0042",
  "line_items": [
    {
      "description": "Pro Plan (Jul 2026)",
      "quantity": 1,
      "unit_amount": 4900,
      "amount": 4900
    }
  ],
  "created_at": "2026-07-01T00:00:00Z"
}
```

### Invoice Status Values

| Status | Description |
|---|---|
| `open` | The invoice has been generated and is awaiting payment. |
| `paid` | Payment has been received in full. |
| `void` | The invoice has been voided and is no longer collectible. |
| `uncollectible` | Payment attempts have been exhausted. The invoice is written off. |

### Payment Terms

| Value | Description |
|---|---|
| `due_on_receipt` | Payment is due immediately when the invoice is generated. |
| `net15` | Payment is due 15 days after the invoice date. |
| `net30` | Payment is due 30 days after the invoice date. |
| `net60` | Payment is due 60 days after the invoice date. |

Payment terms are set on the subscription and inherited by every invoice generated for that subscription.

---

## How Invoices Are Generated

Recurso automatically creates invoices at the start of each billing period:

1. The subscription's `current_period_end` is reached.
2. A new billing period begins (`current_period_start` advances).
3. An invoice is generated with line items reflecting the plan price, any prorations, and applicable taxes.
4. The invoice is emailed to the customer.
5. If a payment method is on file and `payment_terms` is `due_on_receipt`, Recurso attempts to charge the customer immediately.

No API call is needed to trigger invoice generation -- it happens automatically.

---

## List Invoices

Retrieve all invoices for your tenant. You can filter by customer or subscription.

**GET** `/v1/invoices`

```bash
curl https://api.recurso.dev/v1/invoices \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "status": "open",
      "subtotal": 4900,
      "tax_amount": 882,
      "total": 5782,
      "currency": "USD",
      "payment_terms": "net30",
      "due_date": "2026-08-01T00:00:00Z",
      "invoice_number": "INV-2026-0042",
      "created_at": "2026-07-01T00:00:00Z"
    },
    {
      "id": "inv_ee11ff22-3344-5566-7788-99aabbccdd00",
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "subscription_id": "sub_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "status": "paid",
      "subtotal": 4900,
      "tax_amount": 882,
      "total": 5782,
      "currency": "USD",
      "payment_terms": "net30",
      "due_date": "2026-07-01T00:00:00Z",
      "paid_at": "2026-06-18T09:30:00Z",
      "invoice_number": "INV-2026-0041",
      "created_at": "2026-06-01T00:00:00Z"
    }
  ],
  "has_more": false
}
```

---

## Download Invoice PDF

Generate and download a PDF version of an invoice. The response is a binary PDF file.

**GET** `/v1/invoices/:id/pdf`

```bash
curl https://api.recurso.dev/v1/invoices/inv_3c4d5e6f-7a8b-9012-cdef-345678901234/pdf \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  --output invoice-INV-2026-0042.pdf
```

This returns a `Content-Type: application/pdf` response. Use the `--output` flag to save the file locally.

---

## HTML Invoice Preview

Render a preview of the invoice as HTML. This is useful for displaying invoices in your application without downloading a PDF.

**GET** `/v1/invoices/:id/preview`

```bash
curl https://api.recurso.dev/v1/invoices/inv_3c4d5e6f-7a8b-9012-cdef-345678901234/preview \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>Invoice INV-2026-0042</title></head>
  <body>
    <!-- Rendered invoice HTML -->
  </body>
</html>
```

---

## Customer Checkout Flow

Recurso provides a hosted checkout page for customers to pay their invoices. This is useful when customers do not have a payment method on file or when you want to send a payment link.

### Step 1: Get the Checkout Page

Direct your customer to the checkout URL. No authentication is required -- the URL contains a unique invoice token.

**GET** `/checkout/:id`

```
https://api.recurso.dev/checkout/inv_3c4d5e6f-7a8b-9012-cdef-345678901234
```

This renders a hosted payment page where the customer can review the invoice and enter payment details.

### Step 2: Submit Payment

When the customer completes the checkout form, the payment is submitted.

**POST** `/checkout/:id/pay`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `payment_method_token` | string | Yes | Tokenized payment method from the checkout form. |

```bash
curl -X POST https://api.recurso.dev/checkout/inv_3c4d5e6f-7a8b-9012-cdef-345678901234/pay \
  -H "Content-Type: application/json" \
  -d '{
    "payment_method_token": "tok_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
  }'
```

Response:
```json
{
  "status": "paid",
  "invoice_id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
  "amount_paid": 5782,
  "currency": "USD",
  "paid_at": "2026-07-05T16:45:00Z"
}
```

---

## Payment Order Creation

Create a payment order to initiate payment collection for an invoice through your payment gateway.

**POST** `/payments/order`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `invoice_id` | string | Yes | The ID of the invoice to create a payment order for. |

```bash
curl -X POST https://api.recurso.dev/payments/order \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234"
  }'
```

Response:
```json
{
  "id": "order_aabb1234-ccdd-5678-eeff-99001122aabb",
  "invoice_id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
  "amount": 5782,
  "currency": "USD",
  "status": "created",
  "gateway_order_id": "order_Gx7Kf9mPqR2sT4",
  "created_at": "2026-07-05T16:40:00Z"
}
```

Use the `gateway_order_id` to complete the payment through your payment gateway's client-side SDK.

---

## End-to-End Invoice Flow

Here is the typical lifecycle of an invoice:

1. **Generated** -- Recurso creates the invoice at the start of a billing period. Status: `open`.
2. **Delivered** -- The invoice is emailed to the customer with a checkout link.
3. **Payment attempted** -- If a payment method is on file and terms are `due_on_receipt`, Recurso charges automatically.
4. **Paid** -- Payment succeeds. Status: `paid`.
5. **Past due** -- If payment fails or the due date passes, the subscription may move to `past_due`.
6. **Voided** -- An admin voids the invoice if it was created in error. Status: `void`.
7. **Written off** -- After exhausting retries, the invoice is marked `uncollectible`.
