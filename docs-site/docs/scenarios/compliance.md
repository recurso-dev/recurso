---
sidebar_position: 15
---

# Compliance (GST, E-Invoicing, RBI)

India-specific compliance features in Recurso cover three areas: **GST configuration and validation**, **E-Invoicing via the Invoice Registration Portal (IRP)**, and **RBI consent tracking** for recurring payments. This guide walks through each area with API examples.

---

## GST Configuration

### Retrieve GST Settings

Fetch your current GST configuration, including your GSTIN, HSN/SAC codes, and tax rates.

```bash
curl -X GET https://api.recurso.dev/v1/settings/gst \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "gstin": "29AABCU9603R1ZM",
  "legal_name": "Recurso Technologies Pvt Ltd",
  "state_code": "29",
  "hsn_code": "998314",
  "default_tax_rate": 18.0,
  "cgst_rate": 9.0,
  "sgst_rate": 9.0,
  "igst_rate": 18.0,
  "reverse_charge": false,
  "enabled": true,
  "updated_at": "2026-03-10T08:22:00Z"
}
```

### Update GST Settings

Update your GSTIN, HSN code, or tax rates.

```bash
curl -X PUT https://api.recurso.dev/v1/settings/gst \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "gstin": "29AABCU9603R1ZM",
    "hsn_code": "998314",
    "default_tax_rate": 18.0,
    "cgst_rate": 9.0,
    "sgst_rate": 9.0,
    "igst_rate": 18.0,
    "reverse_charge": false
  }'
```

**Response:**

```json
{
  "gstin": "29AABCU9603R1ZM",
  "legal_name": "Recurso Technologies Pvt Ltd",
  "state_code": "29",
  "hsn_code": "998314",
  "default_tax_rate": 18.0,
  "cgst_rate": 9.0,
  "sgst_rate": 9.0,
  "igst_rate": 18.0,
  "reverse_charge": false,
  "enabled": true,
  "updated_at": "2026-06-23T14:05:00Z"
}
```

### Validate GST Configuration

Run a validation check against the GST Network (GSTN) to confirm that your GSTIN is active and details match.

```bash
curl -X POST https://api.recurso.dev/v1/settings/gst/validate \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "valid": true,
  "gstin": "29AABCU9603R1ZM",
  "gstn_status": "Active",
  "legal_name_match": true,
  "state_code_match": true,
  "validated_at": "2026-06-23T14:06:12Z"
}
```

If validation fails, the response includes an `errors` array with details about each mismatch.

---

## E-Invoicing

Recurso integrates with India's Invoice Registration Portal (IRP) to generate e-invoices with a unique Invoice Reference Number (IRN) and signed QR code for every invoice that meets the e-invoicing threshold.

### Retrieve E-Invoice for an Invoice

Fetch the e-invoice record attached to a specific invoice, including the IRN, acknowledgement number, and signed QR code.

```bash
curl -X GET https://api.recurso.dev/v1/invoices/inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d/einvoice \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "einv_c4d5e6f7-89a0-4b1c-de23-f4a5b6c7d8e9",
  "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
  "irn": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
  "ack_number": "132910000012345",
  "ack_date": "2026-06-20T10:30:00Z",
  "status": "generated",
  "signed_qr_code": "eyJhbGciOiJSUzI1NiIsInR5cCI6Ikp...",
  "signed_invoice_url": "https://api.recurso.dev/v1/invoices/inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d/einvoice/pdf",
  "created_at": "2026-06-20T10:30:00Z"
}
```

### Retry E-Invoice Generation

If e-invoice generation failed (for example, due to a temporary IRP outage), retry it.

```bash
curl -X POST https://api.recurso.dev/v1/invoices/inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d/einvoice/retry \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "einv_c4d5e6f7-89a0-4b1c-de23-f4a5b6c7d8e9",
  "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
  "status": "generated",
  "irn": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
  "ack_number": "132910000012346",
  "ack_date": "2026-06-23T14:10:00Z",
  "created_at": "2026-06-23T14:10:00Z"
}
```

### Cancel E-Invoice

Cancel a previously generated e-invoice. This is only permitted within 24 hours of generation, as per IRP rules.

```bash
curl -X POST https://api.recurso.dev/v1/invoices/inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d/einvoice/cancel \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "einv_c4d5e6f7-89a0-4b1c-de23-f4a5b6c7d8e9",
  "invoice_id": "inv_8f3a2b1c-47e9-4d6a-bc12-9e8f7a6b5c4d",
  "irn": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
  "status": "cancelled",
  "cancel_date": "2026-06-23T14:15:00Z"
}
```

---

## IRP Configuration

The Invoice Registration Portal (IRP) settings control which IRP provider Recurso connects to and with what credentials.

### Retrieve IRP Settings

```bash
curl -X GET https://api.recurso.dev/v1/settings/irp \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "provider": "nic",
  "environment": "production",
  "username": "recurso_api_user",
  "client_id": "ABCD1234",
  "enabled": true,
  "last_connected_at": "2026-06-22T18:00:00Z",
  "updated_at": "2026-05-01T09:00:00Z"
}
```

### Update IRP Settings

```bash
curl -X PUT https://api.recurso.dev/v1/settings/irp \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "nic",
    "environment": "production",
    "username": "recurso_api_user",
    "password": "your_irp_password",
    "client_id": "ABCD1234",
    "client_secret": "s3cretK3y",
    "enabled": true
  }'
```

**Response:**

```json
{
  "provider": "nic",
  "environment": "production",
  "username": "recurso_api_user",
  "client_id": "ABCD1234",
  "enabled": true,
  "updated_at": "2026-06-23T14:20:00Z"
}
```

### Test IRP Connection

Verify that Recurso can successfully authenticate with the IRP using your credentials.

```bash
curl -X POST https://api.recurso.dev/v1/settings/irp/test \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "success": true,
  "provider": "nic",
  "environment": "production",
  "latency_ms": 342,
  "tested_at": "2026-06-23T14:21:00Z"
}
```

---

## RBI Consent Tracking

The Reserve Bank of India (RBI) mandates that businesses collecting recurring payments must obtain **explicit customer consent** before each debit and send **pre-debit notifications** at least 24 hours in advance. Recurso tracks consent records and automates pre-debit notification delivery.

### How It Works

1. **Collect consent** -- When a customer subscribes, record their explicit consent via the API.
2. **Pre-debit notifications** -- Recurso automatically sends an SMS and email notification to the customer 24-48 hours before each billing cycle debit, as required by RBI.
3. **Consent revocation** -- If a customer revokes consent, Recurso halts future auto-debits for that subscription and marks the consent as revoked.

### Create a Consent Record

```bash
curl -X POST https://api.recurso.dev/v1/consents \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
    "consent_type": "recurring_debit",
    "channel": "upi",
    "max_amount": 999900,
    "frequency": "monthly",
    "valid_from": "2026-06-23T00:00:00Z",
    "valid_until": "2027-06-23T00:00:00Z"
  }'
```

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `customer_id` | string | Yes | The customer granting consent. |
| `subscription_id` | string | Yes | The subscription this consent applies to. |
| `consent_type` | string | Yes | Type of consent. Supported: `recurring_debit`, `mandate`, `e_nach`. |
| `channel` | string | No | Payment channel: `upi`, `card`, `nach`. |
| `max_amount` | integer | No | Maximum debit amount in paise per cycle. |
| `frequency` | string | No | Billing frequency: `weekly`, `monthly`, `quarterly`, `yearly`. |
| `valid_from` | string | No | Consent start date (ISO 8601). |
| `valid_until` | string | No | Consent expiry date (ISO 8601). |

**Response:**

```json
{
  "id": "cns_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "consent_type": "recurring_debit",
  "channel": "upi",
  "max_amount": 999900,
  "frequency": "monthly",
  "status": "active",
  "valid_from": "2026-06-23T00:00:00Z",
  "valid_until": "2027-06-23T00:00:00Z",
  "created_at": "2026-06-23T14:25:00Z"
}
```

### Revoke Consent

Revoke an active consent. This immediately stops future auto-debits for the associated subscription.

```bash
curl -X POST https://api.recurso.dev/v1/consents/revoke \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{
    "consent_id": "cns_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
    "reason": "customer_requested"
  }'
```

**Response:**

```json
{
  "id": "cns_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
  "status": "revoked",
  "revoked_at": "2026-06-23T14:30:00Z",
  "reason": "customer_requested"
}
```

### List Consents for a Customer

```bash
curl -X GET https://api.recurso.dev/v1/customers/cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890/consents \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "data": [
    {
      "id": "cns_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
      "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
      "consent_type": "recurring_debit",
      "channel": "upi",
      "status": "active",
      "valid_from": "2026-06-23T00:00:00Z",
      "valid_until": "2027-06-23T00:00:00Z",
      "created_at": "2026-06-23T14:25:00Z"
    }
  ],
  "has_more": false
}
```

### Get Consent for a Subscription

Retrieve the current consent record for a specific subscription.

```bash
curl -X GET https://api.recurso.dev/v1/subscriptions/sub_f0e1d2c3-b4a5-6789-0fed-cba987654321/consent \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json"
```

**Response:**

```json
{
  "id": "cns_d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
  "customer_id": "cust_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "subscription_id": "sub_f0e1d2c3-b4a5-6789-0fed-cba987654321",
  "consent_type": "recurring_debit",
  "channel": "upi",
  "max_amount": 999900,
  "frequency": "monthly",
  "status": "active",
  "valid_from": "2026-06-23T00:00:00Z",
  "valid_until": "2027-06-23T00:00:00Z",
  "pre_debit_notification_sent_at": "2026-06-21T10:00:00Z",
  "next_debit_date": "2026-07-23T00:00:00Z",
  "created_at": "2026-06-23T14:25:00Z"
}
```

---

## Pre-Debit Notification Flow

Recurso automates the RBI-mandated pre-debit notification process:

1. **48 hours before billing** -- Recurso sends an SMS and email to the customer with the upcoming debit amount and date.
2. **24 hours before billing** -- If the customer has not opted out, a reminder notification is sent.
3. **On billing date** -- If consent is still active, Recurso initiates the auto-debit. If the customer revoked consent in the interim, the debit is skipped and the subscription is flagged for manual action.

Recurso fires the following webhook events during this process:

| Event | Description |
|---|---|
| `consent.pre_debit_notification_sent` | Pre-debit notification was delivered to the customer. |
| `consent.revoked` | Customer revoked consent; auto-debit is halted. |
| `consent.debit_skipped` | Debit was skipped because consent was revoked or expired. |
