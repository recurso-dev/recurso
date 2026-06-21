# Invoices

Invoices are statements of amounts owed by a customer.

## The Invoice Object

```json
{
  "id": "inv_999",
  "customer_id": "cust_123",
  "subscription_id": "sub_456",
  "amount_due": 5000,
  "amount_paid": 5000,
  "status": "paid",
  "invoice_pdf": "https://api.recurso.dev/invoices/inv_999.pdf"
}
```

## Retrieve an Invoice

**GET** `/invoices/{id}`

Retrieves the details of an invoice.
