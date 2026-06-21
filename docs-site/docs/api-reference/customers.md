# Customers

Customers represent businesses or people who purchase your products.

## The Customer Object

```json
{
  "id": "cust_123",
  "name": "Acme Corp",
  "email": "billing@acme.com",
  "billing_address": {
    "line1": "123 Main St",
    "city": "San Francisco",
    "state": "CA",
    "country": "US",
    "zip": "94105"
  },
  "created_at": "2024-01-01T12:00:00Z"
}
```

## Create a Customer

Creates a new customer.

**POST** `/customers`

| Body Parameter | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | The customer's full name or business name. |
| `email` | string | Yes | The customer's email address for billing. |
| `billing_address` | object | No | The customer's address. |

## Retrieve a Customer

Retrieves the details of an existing customer.

**GET** `/customers/{id}`

| Path Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | string | Yes | The ID of the customer to retrieve. |
