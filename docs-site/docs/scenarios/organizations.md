---
sidebar_position: 10
---

# Organizations & Multi-Entity

Organizations in Recurso allow you to group multiple Tenants under a single umbrella for consolidated reporting and management. This is useful for companies that operate multiple brands, business units, or geographic entities -- each with its own Tenant -- but need a unified view of metrics like MRR.

## The Multi-Entity Model

- A **Tenant** is the fundamental unit in Recurso. Each Tenant has its own customers, subscriptions, plans, invoices, and API keys.
- An **Organization** groups one or more Tenants. It provides consolidated analytics across all member Tenants.
- A Tenant can belong to multiple Organizations (e.g., a regional entity might roll up into both a geographic org and a product-line org).

## The Organization Object

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "Acme Global Holdings",
  "owner_email": "admin@acme-global.com",
  "created_at": "2026-06-15T09:00:00Z",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

## Create an Organization

**POST** `/v1/organizations`

```bash
curl -X POST https://api.recurso.dev/v1/organizations \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Global Holdings",
    "owner_email": "admin@acme-global.com"
  }'
```

Response (`201 Created`):

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "Acme Global Holdings",
  "owner_email": "admin@acme-global.com",
  "created_at": "2026-06-15T09:00:00Z",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

| Body Parameter | Type   | Required | Description                                    |
|----------------|--------|----------|------------------------------------------------|
| `name`         | string | Yes      | Display name for the organization.             |
| `owner_email`  | string | Yes      | Email address of the organization owner. Must be a valid email. |

## List Organizations

**GET** `/v1/organizations`

```bash
curl https://api.recurso.dev/v1/organizations \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "name": "Acme Global Holdings",
      "owner_email": "admin@acme-global.com",
      "created_at": "2026-06-15T09:00:00Z",
      "updated_at": "2026-06-15T09:00:00Z"
    },
    {
      "id": "fedcba98-7654-3210-fedc-ba9876543210",
      "name": "Acme EMEA",
      "owner_email": "emea-admin@acme-global.com",
      "created_at": "2026-06-16T11:00:00Z",
      "updated_at": "2026-06-16T11:00:00Z"
    }
  ]
}
```

## Get an Organization

**GET** `/v1/organizations/:id`

```bash
curl https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "name": "Acme Global Holdings",
    "owner_email": "admin@acme-global.com",
    "created_at": "2026-06-15T09:00:00Z",
    "updated_at": "2026-06-15T09:00:00Z"
  }
}
```

## Update an Organization

**PUT** `/v1/organizations/:id`

```bash
curl -X PUT https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Worldwide Holdings",
    "owner_email": "cfo@acme-global.com"
  }'
```

Response (`200 OK`):

```json
{
  "data": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "name": "Acme Worldwide Holdings",
    "owner_email": "cfo@acme-global.com",
    "created_at": "2026-06-15T09:00:00Z",
    "updated_at": "2026-06-23T14:30:00Z"
  }
}
```

| Body Parameter | Type   | Required | Description                     |
|----------------|--------|----------|---------------------------------|
| `name`         | string | No       | Updated organization name.      |
| `owner_email`  | string | No       | Updated owner email address.    |

## Delete an Organization

**DELETE** `/v1/organizations/:id`

```bash
curl -X DELETE https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "status": "deleted"
}
```

Deleting an organization does not delete the member Tenants. It only removes the organizational grouping.

## Managing Tenants

### Add a Tenant to an Organization

**POST** `/v1/organizations/:id/tenants`

```bash
curl -X POST https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/tenants \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  }'
```

Response (`200 OK`):

```json
{
  "status": "added"
}
```

| Body Parameter | Type   | Required | Description                           |
|----------------|--------|----------|---------------------------------------|
| `tenant_id`    | string | Yes      | UUID of the Tenant to add to the org. |

### List Tenants in an Organization

**GET** `/v1/organizations/:id/tenants`

```bash
curl https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/tenants \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "name": "Acme US",
      "email": "billing@acme-us.com",
      "created_at": "2026-01-10T08:00:00Z"
    },
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "Acme UK",
      "email": "billing@acme-uk.com",
      "created_at": "2026-02-15T10:00:00Z"
    }
  ]
}
```

### Remove a Tenant from an Organization

**DELETE** `/v1/organizations/:id/tenants/:tenant_id`

```bash
curl -X DELETE https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/tenants/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "status": "removed"
}
```

## Consolidated Analytics

### Consolidated MRR

**GET** `/v1/organizations/:id/analytics/mrr`

Returns the combined MRR across all Tenants in the organization, grouped by currency.

```bash
curl https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/analytics/mrr \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

Response (`200 OK`):

```json
{
  "data": [
    {
      "currency": "USD",
      "mrr": 12450000,
      "subscription_count": 347
    },
    {
      "currency": "GBP",
      "mrr": 3280000,
      "subscription_count": 89
    },
    {
      "currency": "INR",
      "mrr": 8750000,
      "subscription_count": 215
    }
  ]
}
```

The `mrr` field is in the smallest currency unit (cents for USD, pence for GBP, paise for INR). Results are grouped by currency because amounts in different currencies cannot be meaningfully summed without an exchange rate conversion. Use the Recurso FX service or your own conversion logic if you need a single consolidated figure.

## Example: Multi-Brand Setup

Suppose you run two SaaS products -- "Acme CRM" and "Acme Analytics" -- each with its own Tenant. You want a unified dashboard for leadership.

**Step 1**: Create an Organization.

```bash
curl -X POST https://api.recurso.dev/v1/organizations \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Software Group",
    "owner_email": "cfo@acme.com"
  }'
```

**Step 2**: Add both Tenants.

```bash
curl -X POST https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/tenants \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"}'
```

```bash
curl -X POST https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/tenants \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'
```

**Step 3**: Pull consolidated MRR.

```bash
curl https://api.recurso.dev/v1/organizations/01234567-89ab-cdef-0123-456789abcdef/analytics/mrr \
  -H "Authorization: Bearer sk_test_9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
```

The response aggregates MRR from both the CRM and Analytics tenants, grouped by currency.
