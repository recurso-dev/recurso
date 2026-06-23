---
sidebar_position: 19
---

# Offline Payments & Virtual Accounts

Enterprise customers often pay via bank transfer, cheque, or cash rather than automated card or UPI payments. Recurso supports these workflows through **virtual accounts** (dedicated bank accounts for each customer) and **offline payment recording** to reconcile invoices manually.

---

## Virtual Accounts

A virtual account is a unique bank account number assigned to a specific customer. When the customer sends a bank transfer to that account, Recurso automatically matches the incoming payment to the customer and their open invoices.

### Create a Virtual Account

```bash
curl -X POST https://api.recurso.dev/v1/virtual-accounts \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "description": "Dedicated account for Acme Corp"
  }'
```

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The customer this virtual account is assigned to. |
| `description` | string | No | A note for internal reference. |

**Response:**

```json
{
  "id": "va_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "account_number": "9876540001234567",
  "ifsc_code": "RATN0VAAPIS",
  "bank_name": "RBL Bank",
  "beneficiary_name": "Recurso Technologies Pvt Ltd",
  "status": "active",
  "description": "Dedicated account for Acme Corp",
  "total_received": 0,
  "created_at": "2026-06-23T15:25:00Z"
}
```

Share the `account_number` and `ifsc_code` with your customer. Any NEFT, RTGS, or IMPS transfer to this account will be attributed to the customer automatically.

### List Virtual Accounts

```bash
curl -X GET "https://api.recurso.dev/v1/virtual-accounts?limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "va_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
      "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Acme Corp",
      "account_number": "9876540001234567",
      "ifsc_code": "RATN0VAAPIS",
      "bank_name": "RBL Bank",
      "status": "active",
      "total_received": 4999900,
      "last_payment_at": "2026-06-15T09:30:00Z",
      "created_at": "2026-06-23T15:25:00Z"
    },
    {
      "id": "va_e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b",
      "customer_id": "cust_b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "customer_name": "Globex Industries",
      "account_number": "9876540009876543",
      "ifsc_code": "RATN0VAAPIS",
      "bank_name": "RBL Bank",
      "status": "active",
      "total_received": 14999700,
      "last_payment_at": "2026-06-20T14:15:00Z",
      "created_at": "2026-04-10T11:00:00Z"
    }
  ],
  "has_more": false
}
```

---

## Record an Offline Payment

When a customer pays by bank transfer (outside of a virtual account), cheque, or cash, record the payment manually to mark the associated invoice as paid.

```bash
curl -X POST https://api.recurso.dev/v1/payments/offline \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
    "amount": 4999900,
    "method": "bank_transfer",
    "reference": "NEFT-REF-2026062300012345",
    "received_at": "2026-06-22T10:00:00Z",
    "notes": "Payment received via NEFT from Acme Corp HDFC account"
  }'
```

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `invoice_id` | string | Yes | The invoice this payment applies to. |
| `amount` | integer | Yes | Amount received in the smallest currency unit (paise for INR, cents for USD). |
| `method` | string | Yes | Payment method: `bank_transfer`, `cheque`, or `cash`. |
| `reference` | string | Yes | External reference number (NEFT ref, cheque number, receipt ID). |
| `received_at` | string | No | When the payment was received (ISO 8601). Defaults to now. |
| `notes` | string | No | Internal notes about the payment. |

**Response:**

```json
{
  "id": "pay_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
  "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "amount": 4999900,
  "method": "bank_transfer",
  "reference": "NEFT-REF-2026062300012345",
  "status": "recorded",
  "invoice_status": "paid",
  "received_at": "2026-06-22T10:00:00Z",
  "notes": "Payment received via NEFT from Acme Corp HDFC account",
  "created_at": "2026-06-23T15:30:00Z"
}
```

Recording an offline payment automatically updates the invoice status. If the amount matches the invoice total, the invoice is marked as `paid`. If the amount is less, the invoice is marked as `partially_paid` with the remaining balance tracked.

### List Offline Payments

```bash
curl -X GET "https://api.recurso.dev/v1/payments/offline?limit=10" \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "pay_f6a7b8c9-d0e1-2f3a-4b5c-6d7e8f9a0b1c",
      "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
      "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "customer_name": "Acme Corp",
      "amount": 4999900,
      "method": "bank_transfer",
      "reference": "NEFT-REF-2026062300012345",
      "status": "recorded",
      "received_at": "2026-06-22T10:00:00Z",
      "created_at": "2026-06-23T15:30:00Z"
    },
    {
      "id": "pay_a7b8c9d0-e1f2-3a4b-5c6d-7e8f9a0b1c2d",
      "invoice_id": "inv_9a0b1c2d-3e4f-5a6b-7c8d-9e0f1a2b3c4d",
      "customer_id": "cust_b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "customer_name": "Globex Industries",
      "amount": 14999700,
      "method": "cheque",
      "reference": "CHQ-445566",
      "status": "recorded",
      "received_at": "2026-06-20T14:15:00Z",
      "created_at": "2026-06-20T16:00:00Z"
    }
  ],
  "has_more": false
}
```

---

## Workflow: Enterprise Bank Transfer Billing

Here is a typical end-to-end workflow for enterprise customers who pay via bank transfer:

1. **Create a virtual account** for the customer using `POST /v1/virtual-accounts`.
2. **Share the account details** (account number, IFSC, beneficiary name) on the invoice PDF or via email.
3. **Generate an invoice** for the subscription cycle. The invoice will show a `pending` status.
4. **Customer transfers funds** to the virtual account via NEFT/RTGS/IMPS.
5. **Recurso detects the incoming transfer** and automatically matches it to the customer. If the amount matches an open invoice, the invoice is marked as `paid` and a `payment.received` webhook is fired.
6. **If auto-matching fails** (e.g., the transfer amount does not match any open invoice), the payment appears as `unmatched` in your dashboard. You can then manually record it using `POST /v1/payments/offline`.

---

## Partial Payments

If a customer sends less than the full invoice amount, Recurso handles it as a partial payment:

- The invoice status becomes `partially_paid`.
- The `amount_remaining` field on the invoice reflects the outstanding balance.
- You can record additional offline payments against the same invoice until the full amount is covered.

```bash
curl -X POST https://api.recurso.dev/v1/payments/offline \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
    "amount": 2500000,
    "method": "bank_transfer",
    "reference": "NEFT-REF-2026062300067890"
  }'
```

**Response:**

```json
{
  "id": "pay_b8c9d0e1-f2a3-4b5c-6d7e-8f9a0b1c2d3e",
  "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
  "amount": 2500000,
  "method": "bank_transfer",
  "reference": "NEFT-REF-2026062300067890",
  "status": "recorded",
  "invoice_status": "partially_paid",
  "invoice_amount_remaining": 2499900,
  "received_at": "2026-06-23T15:35:00Z",
  "created_at": "2026-06-23T15:35:00Z"
}
```
