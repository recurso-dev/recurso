---
sidebar_position: 21
---

# Analytics & Ledger

Recurso provides built-in analytics for tracking key subscription metrics, a double-entry ledger for financial record-keeping, and developer tools for managing API keys and account settings. This guide covers all three areas.

---

## Analytics

### Monthly Recurring Revenue (MRR)

Retrieve your current MRR and historical MRR data. MRR is calculated as the sum of all active subscription amounts, normalized to a monthly value.

**GET** `/v1/analytics/mrr`

```bash
curl https://api.recurso.dev/v1/analytics/mrr \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "current_mrr": 1247600,
  "currency": "USD",
  "period": "2026-06",
  "history": [
    {
      "period": "2026-06",
      "mrr": 1247600,
      "new_mrr": 147000,
      "churned_mrr": 49000,
      "expansion_mrr": 24500,
      "net_new_mrr": 122500
    },
    {
      "period": "2026-05",
      "mrr": 1125100,
      "new_mrr": 196000,
      "churned_mrr": 73500,
      "expansion_mrr": 0,
      "net_new_mrr": 122500
    },
    {
      "period": "2026-04",
      "mrr": 1002600,
      "new_mrr": 245000,
      "churned_mrr": 24500,
      "expansion_mrr": 49000,
      "net_new_mrr": 269500
    }
  ]
}
```

### MRR Fields

| Field | Type | Description |
|---|---|---|
| `current_mrr` | integer | Total MRR in cents for the current period. |
| `new_mrr` | integer | MRR from new subscriptions created in this period. |
| `churned_mrr` | integer | MRR lost from cancelled subscriptions. |
| `expansion_mrr` | integer | MRR gained from plan upgrades or quantity increases. |
| `net_new_mrr` | integer | Net change: `new_mrr + expansion_mrr - churned_mrr`. |

---

### Usage Statistics

Retrieve aggregated usage statistics across your tenant. This includes counts of active subscriptions, total customers, open invoices, and more.

**GET** `/v1/analytics/usage`

```bash
curl https://api.recurso.dev/v1/analytics/usage \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "period": "2026-06",
  "active_subscriptions": 87,
  "total_customers": 142,
  "new_customers_this_period": 12,
  "open_invoices": 15,
  "paid_invoices_this_period": 74,
  "total_revenue_this_period": 1185400,
  "currency": "USD",
  "churn_rate": 3.92,
  "average_revenue_per_customer": 14339
}
```

| Field | Type | Description |
|---|---|---|
| `active_subscriptions` | integer | Number of subscriptions in `active` status. |
| `total_customers` | integer | Total customer count. |
| `new_customers_this_period` | integer | Customers created in the current billing period. |
| `open_invoices` | integer | Invoices in `open` status awaiting payment. |
| `paid_invoices_this_period` | integer | Invoices paid in the current period. |
| `total_revenue_this_period` | integer | Total revenue collected this period in cents. |
| `churn_rate` | number | Percentage of customers who cancelled this period. |
| `average_revenue_per_customer` | integer | ARPC in cents (`total_revenue / total_customers`). |

---

### GenAI Query

Ask natural language questions about your billing data. Recurso uses a GenAI model to interpret your question, query the underlying data, and return a human-readable answer.

**POST** `/v1/analytics/ask`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `question` | string | Yes | A natural language question about your billing data. |

```bash
curl -X POST https://api.recurso.dev/v1/analytics/ask \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Which customers have the highest lifetime value?"
  }'
```

Response:
```json
{
  "question": "Which customers have the highest lifetime value?",
  "answer": "Your top 3 customers by lifetime value are: 1) Acme Corp ($14,694.00 across 18 invoices), 2) TechStart India Pvt Ltd ($11,997.00 across 12 invoices), 3) GlobalSync Ltd ($8,442.00 across 9 invoices).",
  "data": [
    {
      "customer_id": "cust_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "customer_name": "Acme Corp",
      "lifetime_value": 1469400,
      "invoice_count": 18
    },
    {
      "customer_id": "cust_bb2233cc-dd44-5566-ee77-ff8899001122",
      "customer_name": "TechStart India Pvt Ltd",
      "lifetime_value": 1199700,
      "invoice_count": 12
    },
    {
      "customer_id": "cust_d4e5f6a7-b8c9-0123-def0-456789012345",
      "customer_name": "GlobalSync Ltd",
      "lifetime_value": 844200,
      "invoice_count": 9
    }
  ],
  "generated_at": "2026-06-23T15:00:00Z"
}
```

Example questions you can ask:

- "What is my MRR growth rate over the last 6 months?"
- "How many customers are on the Pro Plan?"
- "Which invoices are overdue by more than 30 days?"
- "What is the average time to payment for invoices this quarter?"

---

## Ledger

Recurso maintains a double-entry accounting ledger that records every financial transaction. Every invoice, payment, credit note, and refund creates corresponding debit and credit entries, ensuring your books always balance.

### Double-Entry Accounting Model

Every financial event in Recurso creates at least two ledger entries:

| Event | Debit Account | Credit Account |
|---|---|---|
| Invoice created | Accounts Receivable (AR) | Revenue |
| Payment received | Cash / Bank | Accounts Receivable (AR) |
| Credit note issued | Revenue | Accounts Receivable (AR) |
| Refund processed | Accounts Receivable (AR) | Cash / Bank |

For example, when Recurso generates a $49.00 invoice:
- **Debit** Accounts Receivable: $49.00 (the customer owes you money)
- **Credit** Revenue: $49.00 (you earned revenue)

When the customer pays:
- **Debit** Cash: $49.00 (money received)
- **Credit** Accounts Receivable: $49.00 (the customer no longer owes)

### List Ledger Accounts

Retrieve all ledger accounts for your tenant.

**GET** `/v1/ledger/accounts`

```bash
curl https://api.recurso.dev/v1/ledger/accounts \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "acct_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Accounts Receivable",
      "type": "asset",
      "code": "1200",
      "balance": 578200,
      "currency": "USD"
    },
    {
      "id": "acct_2b3c4d5e-6f7a-8901-bcde-f12345678901",
      "name": "Revenue",
      "type": "revenue",
      "code": "4000",
      "balance": 1185400,
      "currency": "USD"
    },
    {
      "id": "acct_3c4d5e6f-7a8b-9012-cdef-234567890123",
      "name": "Cash",
      "type": "asset",
      "code": "1000",
      "balance": 607200,
      "currency": "USD"
    },
    {
      "id": "acct_4d5e6f7a-8b9c-0123-defa-345678901234",
      "name": "Refunds & Chargebacks",
      "type": "expense",
      "code": "5100",
      "balance": 24500,
      "currency": "USD"
    }
  ],
  "has_more": false
}
```

### Ledger Account Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique ledger account identifier. |
| `name` | string | Human-readable account name. |
| `type` | string | Account type: `asset`, `liability`, `revenue`, or `expense`. |
| `code` | string | Standard accounting code. |
| `balance` | integer | Current account balance in cents. |
| `currency` | string | Currency of the account. |

---

### Get Ledger Entries

Retrieve ledger entries for a specific account. Each entry records a debit or credit with the corresponding transaction reference.

**GET** `/v1/ledger/entries?account_id=...`

```bash
curl "https://api.recurso.dev/v1/ledger/entries?account_id=acct_1a2b3c4d-5e6f-7890-abcd-ef1234567890" \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "entry_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "account_id": "acct_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "type": "debit",
      "amount": 5782,
      "currency": "USD",
      "description": "Invoice INV-2026-0042 created",
      "reference_type": "invoice",
      "reference_id": "inv_3c4d5e6f-7a8b-9012-cdef-345678901234",
      "created_at": "2026-06-23T00:00:00Z"
    },
    {
      "id": "entry_b2c3d4e5-f6a7-8901-bcde-f23456789012",
      "account_id": "acct_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "type": "credit",
      "amount": 5782,
      "currency": "USD",
      "description": "Payment received for INV-2026-0042",
      "reference_type": "payment",
      "reference_id": "pay_c3d4e5f6-a7b8-9012-cdef-345678901234",
      "created_at": "2026-06-18T09:30:00Z"
    },
    {
      "id": "entry_c3d4e5f6-a7b8-9012-cdef-456789012345",
      "account_id": "acct_1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "type": "debit",
      "amount": 5782,
      "currency": "USD",
      "description": "Invoice INV-2026-0043 created",
      "reference_type": "invoice",
      "reference_id": "inv_d4e5f6a7-b8c9-0123-defa-567890123456",
      "created_at": "2026-06-23T00:00:01Z"
    }
  ],
  "has_more": true
}
```

### Ledger Entry Fields

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique entry identifier. |
| `account_id` | string | The ledger account this entry belongs to. |
| `type` | string | Either `debit` or `credit`. |
| `amount` | integer | Amount in cents. |
| `description` | string | Human-readable description of the transaction. |
| `reference_type` | string | The type of object that created this entry (e.g., `invoice`, `payment`, `credit_note`). |
| `reference_id` | string | The ID of the referenced object. |
| `created_at` | string | ISO 8601 timestamp. |

---

## Developer Settings

Manage your API keys and account settings programmatically.

### List API Keys

Retrieve all API keys for your account. Secret keys are partially redacted in the response.

**GET** `/v1/developer/keys`

```bash
curl https://api.recurso.dev/v1/developer/keys \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "data": [
    {
      "id": "key_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "Production Key",
      "prefix": "sk_live_",
      "last4": "7890",
      "environment": "live",
      "created_at": "2026-01-15T09:00:00Z",
      "last_used_at": "2026-06-23T14:22:00Z"
    },
    {
      "id": "key_b2c3d4e5-f6a7-8901-bcde-f23456789012",
      "name": "Test Key",
      "prefix": "sk_test_",
      "last4": "7890",
      "environment": "test",
      "created_at": "2026-01-15T09:00:00Z",
      "last_used_at": "2026-06-23T15:00:00Z"
    }
  ],
  "has_more": false
}
```

### Create an API Key

Generate a new API key. The full secret is only returned once at creation time -- store it securely.

**POST** `/v1/developer/keys`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | A human-readable name for the key. |
| `environment` | string | Yes | Either `test` or `live`. |

```bash
curl -X POST https://api.recurso.dev/v1/developer/keys \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Pipeline Key",
    "environment": "test"
  }'
```

Response:
```json
{
  "id": "key_c3d4e5f6-a7b8-9012-cdef-345678901234",
  "name": "CI/CD Pipeline Key",
  "secret": "sk_test_9f8e7d6c5b4a32100fedcba0987654321abcdef",
  "prefix": "sk_test_",
  "last4": "cdef",
  "environment": "test",
  "created_at": "2026-06-23T15:10:00Z"
}
```

The `secret` field contains the full API key. This is the only time it will be returned in full. Store it in a secure location such as a secrets manager or environment variable.

---

### Get Account Details

Retrieve your account information including the tenant name, billing email, and configuration.

**GET** `/v1/account`

```bash
curl https://api.recurso.dev/v1/account \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890"
```

Response:
```json
{
  "id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "name": "Acme SaaS Inc.",
  "email": "admin@acme-saas.com",
  "default_currency": "USD",
  "timezone": "America/New_York",
  "webhook_url": "https://example.com/webhooks/recurso",
  "tax_enabled": true,
  "created_at": "2026-01-01T00:00:00Z"
}
```

### Update Account Settings

Update your account configuration.

**PUT** `/v1/account`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Your business or tenant name. |
| `email` | string | No | Primary admin email address. |
| `default_currency` | string | No | Default currency for new plans and invoices. |
| `timezone` | string | No | IANA timezone string (e.g., `America/New_York`). |
| `tax_enabled` | boolean | No | Whether to enable automatic tax calculation on invoices. |

```bash
curl -X PUT https://api.recurso.dev/v1/account \
  -H "Authorization: Bearer sk_test_51a2b3c4d5e6f7890abcdef1234567890" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme SaaS Inc.",
    "default_currency": "USD",
    "timezone": "America/New_York",
    "tax_enabled": true
  }'
```

Response:
```json
{
  "id": "ten_f8e7d6c5-b4a3-2190-fedc-ba0987654321",
  "name": "Acme SaaS Inc.",
  "email": "admin@acme-saas.com",
  "default_currency": "USD",
  "timezone": "America/New_York",
  "webhook_url": "https://example.com/webhooks/recurso",
  "tax_enabled": true,
  "created_at": "2026-01-01T00:00:00Z"
}
```

---

## Best Practices

- **Use MRR analytics to track growth.** Monitor `net_new_mrr` each month to understand whether your business is growing, flat, or contracting.
- **Export ledger entries for reconciliation.** Use the ledger entries endpoint to reconcile Recurso data with your accounting software (QuickBooks, Xero, etc.).
- **Rotate API keys periodically.** Create a new key, update your integrations, then revoke the old key.
- **Never expose live keys.** Use `sk_test_` keys for development and testing. Only use `sk_live_` keys in production environments with proper secret management.
- **Leverage the GenAI query for ad-hoc analysis.** Instead of building custom reports, use the `/v1/analytics/ask` endpoint to get quick answers to business questions.
- **Understand the double-entry model.** Every invoice creates a debit to Accounts Receivable and a credit to Revenue. When payment is received, Cash is debited and AR is credited. This ensures your books always balance.
