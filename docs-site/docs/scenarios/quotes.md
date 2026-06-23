---
sidebar_position: 14
---

# Quotes & Proposals

Quotes let you send formal pricing proposals to prospective or existing customers before committing to a subscription or invoice. This guide covers the full quote lifecycle -- from creation through approval to conversion into a live invoice.

---

## The Quote-to-Cash Flow

1. **Create a quote** with line items, pricing, and an expiry date.
2. **Send the quote** to the customer via email.
3. **Customer reviews** the quote and either **accepts** or **declines** it.
4. **Convert the accepted quote** into an invoice to collect payment.

This flow gives your sales team a structured way to negotiate pricing, track proposal status, and seamlessly convert closed deals into billable invoices.

---

## The Quote Object

```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "status": "draft",
  "items": [
    {
      "description": "Pro Plan - Annual",
      "quantity": 1,
      "unit_amount": 468000,
      "amount": 468000
    },
    {
      "description": "Onboarding & Setup Fee",
      "quantity": 1,
      "unit_amount": 50000,
      "amount": 50000
    }
  ],
  "subtotal": 518000,
  "currency": "USD",
  "expiry_date": "2026-07-15T23:59:59Z",
  "notes": "Includes dedicated onboarding specialist for the first 30 days.",
  "created_at": "2026-06-23T10:00:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique quote identifier. |
| `customer_id` | string | The customer this quote is for. |
| `status` | string | Current status: `draft`, `sent`, `accepted`, `declined`, `converted`, or `expired`. |
| `items` | array | Line items with description, quantity, unit_amount, and amount (all in cents). |
| `subtotal` | integer | Total of all line items in cents. |
| `currency` | string | Three-letter ISO 4217 currency code. |
| `expiry_date` | string | ISO 8601 date after which the quote is no longer valid. |
| `notes` | string | Optional notes or terms visible to the customer. |
| `created_at` | string | ISO 8601 timestamp of when the quote was created. |

### Quote Status Values

| Status | Description |
|---|---|
| `draft` | The quote has been created but not yet sent to the customer. |
| `sent` | The quote has been emailed to the customer and is awaiting a response. |
| `accepted` | The customer has accepted the quote. |
| `declined` | The customer has declined the quote. |
| `converted` | The accepted quote has been converted into an invoice. |
| `expired` | The quote passed its `expiry_date` without a response. |

---

## Create a Quote

Build a new quote with line items for a customer.

**POST** `/v1/quotes`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The customer to quote. |
| `items` | array | Yes | Array of line item objects, each with `description`, `quantity`, and `unit_amount`. |
| `expiry_date` | string | Yes | ISO 8601 date when the quote expires. |
| `currency` | string | No | Currency code (defaults to tenant's default currency). |
| `notes` | string | No | Additional notes or terms to include on the quote. |

```bash
curl -X POST https://api.recurso.dev/v1/quotes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "items": [
      {
        "description": "Pro Plan - Annual",
        "quantity": 1,
        "unit_amount": 468000
      },
      {
        "description": "Onboarding & Setup Fee",
        "quantity": 1,
        "unit_amount": 50000
      }
    ],
    "expiry_date": "2026-07-15T23:59:59Z",
    "currency": "USD",
    "notes": "Includes dedicated onboarding specialist for the first 30 days."
  }'
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "status": "draft",
  "items": [
    {
      "description": "Pro Plan - Annual",
      "quantity": 1,
      "unit_amount": 468000,
      "amount": 468000
    },
    {
      "description": "Onboarding & Setup Fee",
      "quantity": 1,
      "unit_amount": 50000,
      "amount": 50000
    }
  ],
  "subtotal": 518000,
  "currency": "USD",
  "expiry_date": "2026-07-15T23:59:59Z",
  "notes": "Includes dedicated onboarding specialist for the first 30 days.",
  "created_at": "2026-06-23T10:00:00Z"
}
```

---

## List Quotes

Retrieve all quotes for your tenant.

**GET** `/v1/quotes`

```bash
curl https://api.recurso.dev/v1/quotes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "status": "draft",
      "subtotal": 518000,
      "currency": "USD",
      "expiry_date": "2026-07-15T23:59:59Z",
      "created_at": "2026-06-23T10:00:00Z"
    },
    {
      "id": "qt_b2c3d4e5-f6a7-8901-bcde-f23456789012",
      "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
      "status": "accepted",
      "subtotal": 299400,
      "currency": "USD",
      "expiry_date": "2026-07-01T23:59:59Z",
      "created_at": "2026-06-15T08:30:00Z"
    }
  ],
  "has_more": false
}
```

---

## Get a Quote

Retrieve a single quote by ID.

**GET** `/v1/quotes/:id`

```bash
curl https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "status": "draft",
  "items": [
    {
      "description": "Pro Plan - Annual",
      "quantity": 1,
      "unit_amount": 468000,
      "amount": 468000
    },
    {
      "description": "Onboarding & Setup Fee",
      "quantity": 1,
      "unit_amount": 50000,
      "amount": 50000
    }
  ],
  "subtotal": 518000,
  "currency": "USD",
  "expiry_date": "2026-07-15T23:59:59Z",
  "notes": "Includes dedicated onboarding specialist for the first 30 days.",
  "created_at": "2026-06-23T10:00:00Z"
}
```

---

## Update a Quote

Modify a quote's line items, expiry date, or notes. Only quotes in `draft` status can be updated.

**PUT** `/v1/quotes/:id`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `items` | array | No | Updated array of line items. |
| `expiry_date` | string | No | New expiry date. |
| `notes` | string | No | Updated notes or terms. |

```bash
curl -X PUT https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "description": "Pro Plan - Annual",
        "quantity": 5,
        "unit_amount": 468000
      },
      {
        "description": "Onboarding & Setup Fee",
        "quantity": 1,
        "unit_amount": 50000
      }
    ],
    "notes": "Volume discount applied for 5 seats. Includes onboarding."
  }'
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "status": "draft",
  "items": [
    {
      "description": "Pro Plan - Annual",
      "quantity": 5,
      "unit_amount": 468000,
      "amount": 2340000
    },
    {
      "description": "Onboarding & Setup Fee",
      "quantity": 1,
      "unit_amount": 50000,
      "amount": 50000
    }
  ],
  "subtotal": 2390000,
  "currency": "USD",
  "expiry_date": "2026-07-15T23:59:59Z",
  "notes": "Volume discount applied for 5 seats. Includes onboarding.",
  "created_at": "2026-06-23T10:00:00Z"
}
```

---

## Delete a Quote

Permanently delete a quote. Only quotes in `draft` or `expired` status can be deleted.

**DELETE** `/v1/quotes/:id`

```bash
curl -X DELETE https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "deleted": true
}
```

---

## Send a Quote to the Customer

Send the quote to the customer via email. This changes the quote status from `draft` to `sent`.

**POST** `/v1/quotes/:id/send`

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d/send \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "status": "sent",
  "sent_to": "billing@acme.com",
  "sent_at": "2026-06-23T10:30:00Z"
}
```

The customer receives an email with a link to view, accept, or decline the quote.

---

## Accept a Quote

Mark a quote as accepted. This is typically triggered when the customer clicks "Accept" in the quote email, but can also be called from your backend.

**POST** `/v1/quotes/:id/accept`

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d/accept \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "status": "accepted",
  "accepted_at": "2026-06-24T09:15:00Z"
}
```

---

## Decline a Quote

Mark a quote as declined.

**POST** `/v1/quotes/:id/decline`

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d/decline \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d",
  "status": "declined",
  "declined_at": "2026-06-24T09:15:00Z"
}
```

---

## Convert a Quote to an Invoice

Convert an accepted quote into a live invoice. This creates a new invoice with line items matching the quote and begins the payment collection process.

**POST** `/v1/quotes/:id/convert`

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_b2c3d4e5-f6a7-8901-bcde-f23456789012/convert \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "qt_b2c3d4e5-f6a7-8901-bcde-f23456789012",
  "status": "converted",
  "converted_at": "2026-06-24T10:00:00Z",
  "invoice": {
    "id": "inv_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
    "status": "open",
    "subtotal": 299400,
    "tax_amount": 0,
    "total": 299400,
    "currency": "USD",
    "invoice_number": "INV-2026-0055",
    "line_items": [
      {
        "description": "Enterprise Plan - Monthly (x3 seats)",
        "quantity": 3,
        "unit_amount": 99800,
        "amount": 299400
      }
    ],
    "created_at": "2026-06-24T10:00:00Z"
  }
}
```

Only quotes with status `accepted` can be converted. After conversion, the quote status changes to `converted` and the resulting invoice follows the standard invoice lifecycle (see [Invoicing & Payments](./invoicing-and-payments.md)).

---

## End-to-End Example

Here is a complete quote-to-cash workflow:

### 1. Create the Quote

```bash
curl -X POST https://api.recurso.dev/v1/quotes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
    "items": [
      {
        "description": "Enterprise Plan - Monthly",
        "quantity": 3,
        "unit_amount": 99800
      }
    ],
    "expiry_date": "2026-07-01T23:59:59Z",
    "currency": "USD",
    "notes": "3-seat Enterprise license. 30-day payment terms."
  }'
```

### 2. Send to Customer

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_b2c3d4e5-f6a7-8901-bcde-f23456789012/send \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

### 3. Customer Accepts

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_b2c3d4e5-f6a7-8901-bcde-f23456789012/accept \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

### 4. Convert to Invoice

```bash
curl -X POST https://api.recurso.dev/v1/quotes/qt_b2c3d4e5-f6a7-8901-bcde-f23456789012/convert \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

The invoice is now live and payment collection begins according to the customer's payment terms.

---

## Best Practices

- **Set realistic expiry dates.** Give customers enough time to review (7-30 days is typical) but not so long that pricing becomes stale.
- **Use notes for terms and conditions.** Include payment terms, scope of work, or validity conditions in the `notes` field.
- **Track quote statuses.** Use the list endpoint to monitor which quotes are pending, expired, or need follow-up.
- **Convert promptly after acceptance.** Convert accepted quotes to invoices quickly to maintain momentum in the sales cycle.
- **Only draft quotes can be edited.** Once a quote is sent, create a new quote if pricing needs to change.
