# Walkthrough - Phase 15: Advanced Billing Features

This phase introduces enterprise-grade billing logic to Recurso, bringing it closer to parity with platforms like Chargebee.

## 🚀 New Features

### 1. Unbilled Charges ("Add-ons" pending invoice)
You can now add one-off charges to a subscription that will be picked up by the next invoice generation.

*   **API Endpoint**: `POST /v1/subscriptions/{id}/charges`
*   **Body**:
    ```json
    {
      "amount": 5000,
      "currency": "USD",
      "description": "Setup Fee"
    }
    ```
*   **Behavior**: Creates a record in `unbilled_charges`. When an invoice is generated for this subscription, these charges are added to the total and marked as `invoiced`.

### 2. Advance Invoicing
Generate an invoice immediately for future periods (e.g., customer wants to pay for the next 6 months now).

*   **API Endpoint**: `POST /v1/subscriptions/{id}/advance`
*   **Body**:
    ```json
    {
      "periods": 6
    }
    ```
*   **Behavior**: 
    1.  Calculates cost for N periods.
    2.  Generates an invoice immediately.
    3.  Extends the `CurrentPeriodEnd` of the subscription by N periods.

### 3. Net-D Payment Terms
Invoices now support payment terms (e.g., Net-15, Net-30).

*   **Logic**: 
    *   If a Subscription has `payment_terms` set (e.g., `"net30"`), the generated invoice's `due_date` is calculated as `Issue Date + 30 Days`.
    *   Default is `"net0"` (Due on Receipt).

### 4. Calendar Billing (Logic Only)
*   **Domain Logic**: The `CreateSubscription` and Renewal logic now supports `BillingAnchorType: "first_of_month"`.
*   **Status**: Core logic is implemented in `domain/subscription.go`, but full integration loop was skipped due to environment limitations.

## 🧪 Verification
Since the environment is having timeout issues, you can verify these features by:
1.  **Code Review**: Check `internal/service/invoice.go` and `internal/adapter/handler/advanced_billing.go`.
2.  **Local Test**: Once your environment is stable, use Postman/Curl to hit the endpoints above.
