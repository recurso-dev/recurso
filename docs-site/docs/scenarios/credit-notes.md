---
sidebar_position: 7
---

# Credit Notes

Credit notes are documents that reduce the amount a customer owes. They are used for refunds, billing error corrections, and goodwill adjustments. This guide covers creating, listing, and understanding when to use credit notes.

## The Credit Note Object

```json
{
  "id": "cn_aabb1122-3344-5566-7788-99eeff001122",
  "tenant_id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "amount": 2500,
  "currency": "USD",
  "reason": "Billing error on invoice INV-2026-0041",
  "status": "issued",
  "invoice_id": "inv_ee11ff22-3344-5566-7788-99aabbccdd00",
  "created_at": "2026-06-23T11:30:00Z"
}
```

### Key Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique credit note identifier. |
| `customer_id` | string | The customer receiving the credit. |
| `amount` | integer | Credit amount in the smallest currency unit (cents). |
| `currency` | string | Three-letter ISO 4217 currency code. |
| `reason` | string | Explanation of why the credit was issued. |
| `status` | string | Status of the credit note (`issued`, `applied`, `voided`). |
| `invoice_id` | string | The invoice this credit note is linked to (if applicable). |
| `created_at` | string | ISO 8601 timestamp of when the credit note was created. |

### Status Values

| Status | Description |
|---|---|
| `issued` | The credit note has been created and is available to apply. |
| `applied` | The credit has been applied to an invoice or the customer's balance. |
| `voided` | The credit note has been canceled and is no longer valid. |

---

## Create a Credit Note

Issue a credit note for a customer.

**POST** `/v1/credit-notes`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The ID of the customer to credit. |
| `amount` | integer | Yes | Credit amount in cents (e.g., `2500` = $25.00). |
| `reason` | string | Yes | A description of why the credit is being issued. |

### Example: Credit for a Billing Error

```bash
curl -X POST https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "amount": 2500,
    "reason": "Billing error on invoice INV-2026-0041"
  }'
```

Response:
```json
{
  "id": "cn_aabb1122-3344-5566-7788-99eeff001122",
  "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "amount": 2500,
  "currency": "USD",
  "reason": "Billing error on invoice INV-2026-0041",
  "status": "issued",
  "invoice_id": null,
  "created_at": "2026-06-23T11:30:00Z"
}
```

### Example: Goodwill Credit

```bash
curl -X POST https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
    "amount": 5000,
    "reason": "Goodwill credit for service disruption on 2026-06-18"
  }'
```

Response:
```json
{
  "id": "cn_ccdd3344-5566-7788-99aa-bbeeff112233",
  "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
  "amount": 5000,
  "currency": "USD",
  "reason": "Goodwill credit for service disruption on 2026-06-18",
  "status": "issued",
  "invoice_id": null,
  "created_at": "2026-06-23T12:00:00Z"
}
```

---

## List Credit Notes

Retrieve all credit notes for your tenant.

**GET** `/v1/credit-notes`

```bash
curl https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "cn_aabb1122-3344-5566-7788-99eeff001122",
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "amount": 2500,
      "currency": "USD",
      "reason": "Billing error on invoice INV-2026-0041",
      "status": "applied",
      "invoice_id": "inv_ee11ff22-3344-5566-7788-99aabbccdd00",
      "created_at": "2026-06-23T11:30:00Z"
    },
    {
      "id": "cn_ccdd3344-5566-7788-99aa-bbeeff112233",
      "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
      "amount": 5000,
      "currency": "USD",
      "reason": "Goodwill credit for service disruption on 2026-06-18",
      "status": "issued",
      "invoice_id": null,
      "created_at": "2026-06-23T12:00:00Z"
    }
  ],
  "has_more": false
}
```

---

## When to Issue Credit Notes

Credit notes are appropriate in several scenarios:

### Refunds

When a customer requests a refund for a paid invoice, issue a credit note for the refund amount. The credit is applied to the customer's balance and offsets future invoices.

```bash
curl -X POST https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "amount": 4900,
    "reason": "Full refund for Pro Plan - June 2026"
  }'
```

### Billing Errors

If an invoice was generated with the wrong amount (e.g., a proration error or duplicate charge), issue a credit note to correct the difference.

```bash
curl -X POST https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
    "amount": 1200,
    "reason": "Correction for duplicate charge on INV-2026-0039"
  }'
```

### Goodwill Adjustments

Offer a credit to retain a customer after a service disruption, missed SLA, or as a loyalty gesture.

```bash
curl -X POST https://api.recurso.dev/v1/credit-notes \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
    "amount": 2000,
    "reason": "Goodwill credit - 3 hours downtime on 2026-06-20"
  }'
```

---

## How Credits Are Applied

Once a credit note is issued:

1. The credit amount is added to the customer's **credit balance**.
2. When the next invoice is generated, Recurso automatically applies available credits to reduce the amount due.
3. If the credit exceeds the invoice total, the remaining balance carries forward to future invoices.
4. The credit note status changes from `issued` to `applied` once the credit has been used.

---

## Best Practices

- **Always include a clear reason.** This creates an audit trail for your finance team.
- **Use credit notes instead of voiding invoices** when partial adjustments are needed. Voiding an invoice removes it entirely, while a credit note preserves the original invoice for record-keeping.
- **Monitor outstanding credits.** Large credit balances may indicate systemic billing issues that need attention.
- **Issue promptly.** Customers expect quick resolution. Issue credit notes as soon as the error or request is confirmed.
